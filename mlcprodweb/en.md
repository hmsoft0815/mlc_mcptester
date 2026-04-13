# MCP-Tester

A powerful command-line tool designed for testing, debugging, and validating Model Context Protocol (MCP) servers based on the latest specifications.

## What it does

MCP-Tester provides developers with a robust environment to ensure their MCP servers are reliable, compliant, and performant. It supports multiple transports, allowing you to test both local stdio processes and remote SSE-based servers. With its integrated scripting engine and server inspector, it automates the tedious parts of server validation.

## Key features

- **Spec-Compliant Validation**: Verify your server's implementation of Tools, Resources, Prompts, and Logging against the official MCP specification.
- **Interactive Scripting Engine**: Create complex test scenarios using a simple, human-readable scripting grammar to automate multi-step workflows.
- **Server Quality Score**: Use the built-in inspector to analyze your server's health, metadata quality, and best practices, receiving an actionable score.
- **Multi-Transport Debugging**: Seamlessly switch between testing local servers via Stdio and remote deployments via SSE.
- **Pagination & Progress Support**: Test advanced features like cursor-based list navigation and real-time progress monitoring.

## Quick Setup

### Claude Desktop
Add the following to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mcp-tester": {
      "command": "mcp-tester",
      "args": ["-server"]
    }
  }
}
```

### Gemini-CLI
Add to your `~/.gemini/settings.json`:

```json
{
  "mcpServers": {
    "mcp-tester": {
      "command": "mcp-tester",
      "args": ["-server"]
    }
  }
}
```
