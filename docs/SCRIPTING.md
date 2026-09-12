# MCP Scripting Engine

The `mcp-tester` scripting engine enables automated test workflows for MCP servers. Scripts are saved in files with the `.mcp` extension.

## General Syntax

- **Commands**: One command per line.
- **Comments**: Lines starting with `#` or `//` are ignored. Trailing comments are also supported.
- **Variables**: Referenced with a `$` prefix (e.g., `$name`). Substitution uses regex matching (`\$([A-Za-z_][A-Za-z0-9_]*)`). Referencing an unknown variable aborts execution with a clear line-referenced error.
- **Strings**: Can be enclosed in double quotes (`"..."`) or single quotes (`'...'`) if they contain spaces or special characters.
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
    - `$.<path>`: Short form for `structuredContent`.

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
