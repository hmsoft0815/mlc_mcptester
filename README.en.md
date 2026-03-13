# MCP-Tester

A command-line tool for testing, debugging, and validating Model Context Protocol (MCP) servers based on the 2025-11-25 specification.

[Deutsche Version](README.md)

## Key Features

- **Multi-Transport**: Supports local processes (`stdio`) and remote servers (`sse`).
- **Full Spec Support**: Tests Tools, Resources (static & templates), Subscriptions, and Prompts.
- **Pagination Support**: Supports cursors for navigating large lists (`list`).
- **Utilities**: Built-in support for Ping, Cancellation, Logging (setLevel), and Progress monitoring.
- **Scripting Engine**: Automated test workflows with variables, type conversion, and assertions.
- **Server Inspector**: Analyzes servers for best practices and provides a Quality Score.
- **Raw Mode**: Bypasses SDK validation for deep-level debugging.
- **Profiles**: Easy management of different server configurations in `mcp-tester.yml`.

---

## The "Everything" Test Server

This project includes a reference server (`cmd/test-server`) that utilizes all features of the MCP protocol (Tools, Resources, Prompts, Logging, Progress, Output Schemas).

---

## Usage

### 1. Installation

**Via Go (Direct from GitHub):**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```
*Note: Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`.*

**Getting Started:**
After installation, you can add your first server and test it immediately:
```bash
# Add a server profile
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# List available tools
mcp-tester tools list -p my-server
```

**Via Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manual Build:**
```bash
task all            # Builds the tester and reference server into the bin/ folder
```

### 2. Commands (Excerpt)

#### Profile Management
Manage different server configurations directly via the CLI:
```bash
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"
mcp-tester profile list
mcp-tester profile disable my-server
mcp-tester profile delete my-server
```

#### Server Inspection
Analyze a server for quality (metadata, prompts, structure):
```bash
# Using a profile from mcp-tester.yml
mcp-tester inspect --profile local

# Direct call without a configuration file
mcp-tester inspect -c "npx -y @modelcontextprotocol/server-everything"
```

#### Tools, Resources & Prompts
```bash
mcp-tester tools list -p local
mcp-tester ping -p local
mcp-tester logging debug -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local
```

#### Test Scripts (Automation)
Execute complex test scenarios:
```bash
./bin/mcp-tester test --script tests/10_cancellation_demo.mcp --profile local -v
```

---

## Documentation

- [The MCP Handbook](buch/README.md) - A comprehensive introduction to MCP (German).
  - [Chapter 4: Tools – The Model's Hands](buch/04_tools_die_haende_des_modells.md)
  - [Chapter 11: Real-time Feedback & Cancellation](buch/11_echtzeit_feedback_und_audio.md)
  - [Chapter 19: Advanced: Long-running Tasks](buch/19_erweiterungen_tasks.md)
  - [Chapter 20: Future: Agentic Servers & Sampling](buch/20_agentische_server_sampling.md)
  - [Chapter 21: User Input & Elicitation](buch/21_benutzerabfragen_elicitation.md)
- [Scripting Reference](docs/SCRIPTING.md) - Detailed documentation of the test grammar.

---

## License
- Code: [MIT](LICENSE)
- Handbook: [CC BY-NC-ND 4.0](buch/LICENSE.md)

---
*Copyright Michael Lechner - 2026-03-09*