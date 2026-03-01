# MCP Scripting Engine

The `mcp-tester` scripting engine enables automated test workflows for MCP servers. Scripts are saved in files with the `.mcp` extension.

## General Syntax

- **Commands**: One command per line.
- **Comments**: Lines starting with `#` or `//` are ignored. Trailing comments are also supported.
- **Variables**: Referenced with a `$` prefix (e.g., `$name`).
- **Strings**: Can be enclosed in double quotes if they contain spaces.

---

## Command Overview

### 1. `call_tool`
Invokes an MCP tool.
```mcp
call_tool <tool_name> [arg1] [arg2] ...
```
- **Arguments**: Can be passed positionally or as named arguments (`key:value`).
    - **Positional**: Arguments are automatically converted to the correct type based on the tool's JSON schema. The order corresponds to the **alphabetical sorting** of the property names in the schema.
    - **Named**: Arguments follow the `key:value` syntax. This is recommended to avoid confusion with alphabetical sorting.
    - **Mixed**: You can mix both; positional arguments will fill the remaining properties in alphabetical order.

### 2. `set_var`
Extracts a value from the last tool response and stores it in a variable.
```mcp
set_var <variable_name> <path>
```
- **Paths**:
    - `rawResponse`: Stores the complete JSON response from the server.
    - `structuredContent.<path>`: Navigates through the JSON structure (dot notation).
    - `$.<path>`: Short form for `structuredContent`.

### 3. `input_var`
Prompts the user for input during the test.
```mcp
input_var <variable_name> ["Interactive Prompt"]
```

### `assert_contains <expected>` or `assert_contains <value> <expected>`
Checks if the last response (text or JSON) or a specific value contains the expected string.
- `assert_contains "Execution finished"` (checks last response)
- `assert_contains $var "expected"` (checks variable content)

---

### `assert_equals <expected>` or `assert_equals <value> <expected>`
Checks for an exact match against the last response or between two values.
- `assert_equals "30"` (checks last response)
- `assert_equals $var "true"` (checks variable content)

---

### 6. `assert_number`
Checks if a value (or a variable) is a valid number.
```mcp
assert_number $variable
```

### 7. `assert_gt`
Checks if the first value is greater than the second.
```mcp
assert_gt $value1 $value2
```
### 8. `assert_string_length`
Checks if the length of a string (or variable) is within a specific range.
```mcp
assert_string_length $variable <min> <max>
```
- `assert_string_length $var 5 10`

---

## Example Script

```mcp
# 1. Call tool and store ID
call_tool create_user "John Doe"
set_var user_id $.id

# 2. Use variable in next call
call_tool get_user $user_id
assert_contains "Doe"

# 3. Mathematical check
set_var score $.profile.score
assert_gt $score 0
```

---

## Execution
A script is started via the `test` menu item or directly via the CLI:
```bash
mcp-tester test --script my_test.mcp --profile my_server
```
