# MCP-Tester

A powerful command-line tool for testing, debugging, and validating Model Context Protocol (MCP) servers.

[Deutsche Version](README.md)

## Key Features

- **Multi-Transport**: Supports local processes (`stdio`) and remote servers (`sse`).
- **Full Spec Support**: Tests Tools, Resources (static & templates), and Prompts.
- **Scripting Engine**: Automated test workflows with variables, type conversion, and assertions.
- **Server Inspector**: Analyzes servers for best practices and provides a Quality Score.
- **Raw Mode**: Bypasses SDK validation for deep-level debugging of non-compliant responses.
- **Profiles**: Easy management of different server configurations in `mcp-tester.yml`.

---

## The "Everything" Test Server

This project includes a reference server (`cmd/test-server`) that utilizes all features of the MCP protocol (Tools, Resources, Prompts, Logging, Progress, Output Schemas).

---

## Usage

### 1. Installation & Build
```bash
task all            # Builds the tester and reference server into the bin/ folder
```

### 2. Commands

#### Server Inspection (NEW)
Analyze a server for quality (metadata, prompts, structure):
```bash
./bin/mcp-tester inspect --profile local
```

#### Tools & Resources
```bash
./bin/mcp-tester list --profile local
./bin/mcp-tester call add --args '{"a": 1, "b": 2}' --profile local
./bin/mcp-tester resources read "mcp://time" --profile local
```

#### Test Scripts (Automation)
Execute complex test scenarios:
```bash
./bin/mcp-tester test --script tests/03_variables_and_math.mcp --profile local
```

---

## Documentation

- [The MCP Handbook](buch/README.md) - A 15-chapter introduction to MCP (German).
- [Integration Testing](INTEGRATION_TESTS.md) - Details on scripting and CI/CD.
- [Prompts Explained](PROMPTS.md) - Everything about instructions and system context.

---

## License
- Code: [MIT](LICENSE)
- Handbook: [CC BY-NC-ND 4.0](buch/LICENSE.md)

---
*Copyright Michael Lechner - 2026-02-28*
