# MCP Scripting Engine

The `mcp-tester` scripting engine enables automated test workflows for MCP servers. Scripts are saved in files with the `.mcp` extension.

## General Syntax

- **Commands**: One command per line.
- **Comments**: Lines starting with `#` or `//` are ignored. Trailing comments are also supported.
- **Variables**: Referenced with a `$` prefix (e.g., `$name`). Substitution uses regex matching (`\$([A-Za-z_][A-Za-z0-9_]*)`). Referencing an unknown variable aborts execution with a clear line-referenced error.
- **Strings**: Can be enclosed in double quotes (`"..."`) or single quotes (`'...'`) if they contain spaces or special characters. Inside double quotes `\"` and `\\` are escapes (`"missing: [\"code\"]"`); every other backslash stays literal. Single quotes are taken verbatim. An unterminated quote fails the line.
- **Lists / Objects**: JSON arrays (`'["a.png", "b.png"]'`) or JSON objects (`'{"key": "value"}'`) can be passed directly as arguments.

---

## Command Overview

### 1. `echo`
Prints a message to the console. Useful for structuring test output and adding section headers in longer scripts.
```mcp
echo "--- Testing user creation workflow ---"
echo "Current ID: $user_id"
```

### 2. `call_tool`
Invokes an MCP tool.
```mcp
call_tool <tool_name> [arg1] [arg2] ...
```
- **Arguments**: Can be passed positionally or as named arguments (`key:value`).
    - **Positional**: Arguments are automatically converted to the correct type based on the tool's JSON schema. The order corresponds to the **alphabetical sorting** of the property names in the schema.
    - **Named**: Arguments follow the `key:value` syntax (e.g. `paths:'["a.png", "b.png"]'`). This is recommended to avoid confusion with alphabetical sorting.
    - **Arrays and Objects**: Supports schemas with `type: "array"`, `type: "object"` as well as nullable definitions (`type: ["null", "array"]`).
    - **Mixed**: You can mix both; positional arguments will fill the remaining properties in alphabetical order.
- **Result check**: If the tool declares an `outputSchema`, the call fails unless the result carries `structuredContent` that matches it — the check strict clients make (the official TypeScript SDK, used by OpenCode, rejects such a call with `-32600`). Results with `isError: true` are exempt. The Go SDK this tester is built on does not check this itself, which is why the tester does.

**Heredoc Support:**
For multiline arguments (e.g. JSON or code blocks), heredoc syntax can be used:
```mcp
call_tool execute_script <<EOF
console.log("Hello from Heredoc!");
console.log(1 + 2);
EOF
```

### 3. `expect_error`
Prepended to a command when an error (tool error or protocol error) is expected.
```mcp
expect_error call_tool add a:"not_a_number"
assert_tool_error
assert_contains "type"

expect_error call_tool unknown_tool
assert_error_code -32602
```

### 4. `assert_tool_error` and `assert_error_code`
The MCP specification strictly distinguishes between application-level tool errors (`isError: true` in the tool result) and JSON-RPC protocol errors (such as invalid request or unknown tool).

- **`assert_tool_error`**: Verifies that the tool returned a result with `isError: true` (alternatively `assert_error_code tool`).
- **`assert_error_code <code>`**: Verifies the numeric JSON-RPC protocol error code of the last failed request. A tool error (`isError: true`) fails on numeric codes.

```mcp
# Tool execution error
expect_error call_tool validate_script script:"invalid"
assert_tool_error

# Protocol error (-32602 = Invalid params / unknown tool)
expect_error call_tool non_existent_tool
assert_error_code -32602
```

#### Common JSON-RPC Error Codes
- `-32700`: Parse error (invalid JSON)
- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params (schema validation on RPC level)
- `-32603`: Internal error

### 5. `set_var`
Extracts a value from the last tool response and stores it in a variable.
```mcp
set_var <variable_name> <path>
```
- **Paths**:
    - `rawResponse`: Stores the complete JSON response from the server.
    - `structuredContent.<path>`: Navigates through the JSON structure (dot notation).
    - `$.<path>`: Short form for `structuredContent.<path>`; if the field is not there, the top level of the result is tried.

### 6. `input_var`
Prompts the user for input during the test.
```mcp
input_var <variable_name> ["Interactive Prompt"]
```

### 7. `assert_contains`
Checks if the last response (text or JSON) or a specific value contains the expected string.
```mcp
assert_contains "Execution finished"
assert_contains $var "expected"
```

### 8. `assert_equals`
Checks for an exact match against the last response or between two values.
```mcp
assert_equals "Result: 30"
assert_equals $var "123"
```

### 9. `assert_number`
Checks if a value (or a variable) is a valid number.
```mcp
assert_number $variable
```

### 10. `assert_gt`
Checks if the first value is greater than the second.
```mcp
assert_gt $value1 $value2
```

### 11. `assert_string_length`
Checks if the length of a string (or variable) is within a specific range.
```mcp
assert_string_length $variable <min> <max>
```

---

### 12. `complete`
Asks the server for argument completions (`completion/complete`).
```mcp
complete <prompt:name|resource:uri> <argument> [value]
```
The result is stored as `{values, total, hasMore}`: `assert_contains` sees the values one per line, `set_var` addresses `values.0` or `total`.
```mcp
complete prompt:persona_developer language g
assert_contains "golang"
set_var first values.0
```

---

### 13. Input requests: `elicit_response`, `sample_response`, `add_root`
Since spec 2026-07-28 a server asks the client for input via *multi round-trip requests*: `tools/call`, `prompts/get` or `resources/read` returns `inputRequests`, the client answers and retries. The tester does this automatically; the script prepares the answers **before** the call. A request without a prepared answer fails the call.
```mcp
elicit_response accept '{"confirm": true}'   # also: decline, cancel
sample_response "The model's reply"          # answer to sampling/createMessage
add_root file:///home/user/project project   # offered via roots/list
```
Answers are used in order (queue). `assert_elicited <text>` checks the message of the last elicitation request, `assert_sampled <text>` the prompt of the last sampling request.
```mcp
elicit_response accept '{"confirm": true}'
call_tool confirm_delete item:"report.pdf"
assert_elicited "report.pdf"
```
On the command line the same answers are given with `--elicit 'accept:{"confirm":true}'`, `--sample "text"` and `--root file:///path` (all repeatable); there an unanswered elicitation is declined.

---

### 14. Notifications: `subscribe`, `wait_notification`
The tester opens a `subscriptions/listen` stream for every list the server marks as `listChanged`, and records `notifications/tools|prompts|resources/list_changed` and `notifications/resources/updated`.
```mcp
subscribe mcp://time                             # resource updates for this URI
wait_notification tools/list_changed             # waits up to 5s (default)
wait_notification resources/updated mcp://time 2s
```
`wait_notification` succeeds with a notification received since the last wait, also one that arrived before the command, and consumes it. With `expect_error`, it checks that nothing arrives.

On the command line, `mcp-tester listen [--subscribe <uri>] [--duration 30s]` prints notifications as they arrive.

---

### 15. Tasks: `call_task`, `start_task`, `wait_task`, `get_task`, `cancel_task`
With the Tasks extension (`io.modelcontextprotocol/tasks`) a server may answer a tool call with a task handle and run it in the background. The tester declares the extension only for these commands; `call_tool` stays synchronous.
```mcp
call_task long_job seconds:2          # start, poll to the end, result as with call_tool
assert_task_status completed

start_task long_job seconds:30        # only the handle
set_var id taskId
get_task $id                          # one tasks/get
cancel_task $id
wait_task $id 5s                      # poll until completed, failed or cancelled
assert_task_status cancelled
```
If a task needs input (status `input_required`), the answers prepared with `elicit_response` / `sample_response` / `add_root` are sent via `tasks/update`. A failed task (`failed`) is an RPC error for `expect_error` / `assert_error_code`. On the command line: `mcp-tester call <tool> --task`.

---

## Example Script

```mcp
echo "--- Starting user workflow ---"

# 1. Call tool and store ID
call_tool create_user "John Doe"
set_var user_id $.id

# 2. Use variable in next call
call_tool get_user $user_id
assert_contains "Doe"

# 3. Pass array argument
call_tool assign_roles user_id:$user_id roles:'["admin", "tester"]'
assert_contains "Roles assigned"

# 4. Mathematical check
set_var score $.profile.score
assert_gt $score 0
```

---

## Execution
A script is started via the `test` menu item or directly via the CLI:
```bash
mcp-tester test --script my_test.mcp --profile my_server
```

The command exits with status 1 as soon as one script command fails, so it can gate CI or a Taskfile. With `--format json`, stdout carries only the summary (including a `failures` list with line and error); per-command output goes to stderr.
