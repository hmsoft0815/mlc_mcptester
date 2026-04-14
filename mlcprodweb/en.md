# MCP-Tester

MCP-Tester is a command-line tool for developers who build MCP servers. It connects to any MCP server — local or remote — and lets you inspect, explore, and validate it directly from the terminal.

No GUI, no browser extension. Just a single binary that speaks the MCP protocol.

## Installation

**Download binary** (no runtime required):

Download the binary for your platform from the download section and place it in your `PATH`.

**Via Go:**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```

**Via curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

## Quick Start: Test any MCP server in 60 seconds

```bash
# Add a server profile (here: the official MCP everything-server via npx)
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# List all tools exposed by the server
mcp-tester tools list -p my-server

# List resources
mcp-tester resources list -p my-server

# Call a tool directly
mcp-tester tools call echo -p my-server --args '{"message": "hello"}'
```

The `-c` flag accepts any command that starts a stdio MCP server. For SSE servers, use `-url`.

## Server Inspection

The `inspect` command analyzes a server for spec compliance, metadata quality, and best practices. It returns a quality score and actionable findings:

```bash
# Inspect via profile
mcp-tester inspect -p my-server

# Inspect directly without a profile
mcp-tester inspect -c "./my-mcp-server"
mcp-tester inspect -url "https://my-server.example.com/sse"
```

The inspector checks: tool descriptions, input schema completeness, resource URI patterns, prompt argument definitions, and more.

## Profile Management

Profiles let you manage multiple servers in a `mcp-tester.yml` file so you don't have to repeat connection flags:

```bash
mcp-tester profile add local -c "./bin/my-server"
mcp-tester profile add remote -url "https://api.example.com/mcp/sse"
mcp-tester profile list
mcp-tester profile disable local
mcp-tester profile delete local
```

## Exploring Tools, Resources & Prompts

```bash
# Tools
mcp-tester tools list -p local
mcp-tester tools call my_tool -p local --args '{"param": "value"}'

# Resources
mcp-tester resources list -p local
mcp-tester resources get "file:///config.yaml" -p local

# Prompts
mcp-tester prompts list -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local

# Utilities
mcp-tester ping -p local
mcp-tester logging debug -p local
```

## Automated Test Scripts

For regression testing and CI integration, MCP-Tester has a scripting engine. Scripts use `.mcp` files with a simple, readable grammar:

```mcp
# Call a tool and store the result
call_tool add a:10 b:20
set_var result $.value

# Assert the outcome
assert_equals $result "30"

# Test with dynamic input
input_var city "Enter city name:"
call_tool get_weather city:$city
assert_contains "temperature"
```

Run a script:
```bash
mcp-tester test --script tests/my-scenario.mcp -p local
mcp-tester test --script tests/my-scenario.mcp -p local -v   # verbose output
```

Scripts support variables, type coercion, assertions (`assert_equals`, `assert_contains`, `assert_number`), and pagination testing via cursors.

## Transport Support

| Transport | Flag | Use case |
|-----------|------|----------|
| stdio | `-c "command"` | Local processes, binaries |
| SSE | `-url "https://..."` | Remote servers, deployed APIs |

Both transports work with all commands: `inspect`, `tools`, `resources`, `prompts`, `test`.

## Bundled Test Server

The download includes `test-server` — a reference implementation that exercises all MCP features (Tools, Resources, Prompts, Logging, Progress, Output Schemas). It is useful for testing your MCP client, not the mcp-tester CLI itself.

```bash
mcp-tester profile add everything -c "test-server"
mcp-tester inspect -p everything
```
