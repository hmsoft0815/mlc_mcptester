# Test Scripting

The MCP-Tester scripting engine allows you to define complex, multi-step test scenarios for your MCP servers. Scripts use the `.mcp` extension and follow a clean, line-based grammar.

## Basic Syntax

- **Commands**: One command per line.
- **Variables**: Reference with `$` (e.g., `$userId`).
- **Comments**: Lines starting with `#` or `//` are ignored.

## Core Commands

### call_tool
Invokes an MCP tool by name.
```mcp
call_tool my_tool param1:value1 param2:value2
```

### set_var
Extracts data from the last response using dot notation.
```mcp
set_var status $.profile.status
```

### assertions
Validate the behavior of your server with built-in checks:
- `assert_contains`: Check if the result includes a specific string.
- `assert_equals`: Check for an exact match.
- `assert_gt`: Check if a number is greater than another value.
- `assert_error_code`: Verify JSON-RPC error codes.

## Example Workflow

```mcp
# Create a user and store the ID
call_tool create_user name:"John"
set_var id $.id

# Verify the user exists
call_tool get_user userId:$id
assert_contains "John"
```
