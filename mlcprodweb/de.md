# MCP-Tester

Ein leistungsstarkes Kommandozeilen-Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern basierend auf den aktuellsten Spezifikationen.

## Was es macht

MCP-Tester bietet Entwicklern eine robuste Umgebung, um sicherzustellen, dass ihre MCP-Server zuverlässig, konform und performant sind. Es unterstützt mehrere Transporte, sodass Sie sowohl lokale Stdio-Prozesse als auch entfernte SSE-basierte Server testen können. Mit der integrierten Scripting-Engine und dem Server-Inspektor automatisiert es die mühsamen Teile der Server-Validierung.

## Wichtigste Funktionen

- **Standardkonforme Validierung**: Überprüfen Sie die Implementierung von Tools, Ressourcen, Prompts und Logging Ihres Servers gegen die offizielle MCP-Spezifikation.
- **Interaktive Scripting-Engine**: Erstellen Sie komplexe Test-Szenarien mit einer einfachen, menschenlesbaren Scripting-Grammatik, um mehrstufige Workflows zu automatisieren.
- **Server Quality Score**: Nutzen Sie den integrierten Inspektor, um die Gesundheit Ihres Servers, die Qualität der Metadaten und Best Practices zu analysieren und einen aussagekräftigen Score zu erhalten.
- **Multi-Transport-Debugging**: Wechseln Sie nahtlos zwischen dem Testen lokaler Server via Stdio und entfernter Deployments via SSE.
- **Unterstützung für Pagination & Progress**: Testen Sie fortgeschrittene Funktionen wie Cursor-basierte Listennavigation und Echtzeit-Fortschrittsüberwachung.

## Schnellstart (Test-Server)

Dieses Paket enthält einen Referenz-Server zum Testen Ihrer Clients.

### Claude Desktop
Fügen Sie Folgendes zu Ihrer `claude_desktop_config.json` hinzu:

```json
{
  "mcpServers": {
    "mcp-test-server": {
      "command": "test-server"
    }
  }
}
```

### Gemini-CLI
Fügen Sie den Server zu Ihrer `~/.gemini/settings.json` hinzu:

```json
{
  "mcpServers": {
    "mcp-test-server": {
      "command": "test-server"
    }
  }
}
```
