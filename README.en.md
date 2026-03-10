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

**Via Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manual Build:**
```bash
task all            # Builds the tester and reference server into the bin/ folder
```

### 2. Commands (Excerpt)

#### Server Inspection
Analyze a server for quality (metadata, prompts, structure):
```bash
./bin/mcp-tester inspect --profile local
```

#### Tools, Resources & Prompts
```bash
./bin/mcp-tester tools list --profile local
./bin/mcp-tester ping --profile local
./bin/mcp-tester logging debug --profile local
./bin/mcp-tester resources templates --profile local
./bin/mcp-tester prompts get code_review --args '{"file_path": "main.go"}'
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