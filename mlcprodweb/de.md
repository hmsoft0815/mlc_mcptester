# MCP-Tester

MCP-Tester ist ein Kommandozeilen-Tool für Entwickler, die MCP-Server bauen. Es verbindet sich mit jedem MCP-Server — lokal oder remote — und erlaubt es, ihn direkt im Terminal zu inspizieren, zu erkunden und zu validieren.

Kein GUI, kein Browser-Plugin. Nur eine einzige Binary, die das MCP-Protokoll spricht.

## Installation

**Binary herunterladen** (kein Runtime erforderlich):

Die Binary für deine Plattform aus dem Download-Bereich herunterladen und in den `PATH` legen.

**Über Go:**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```

**Über curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

## Quick Start: Jeden MCP-Server in 60 Sekunden testen

```bash
# Server-Profil hinzufügen (hier: offizieller MCP-Everything-Server via npx)
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# Alle Tools des Servers auflisten
mcp-tester tools list -p my-server

# Ressourcen auflisten
mcp-tester resources list -p my-server

# Ein Tool direkt aufrufen
mcp-tester tools call echo -p my-server --args '{"message": "hallo"}'
```

Das `-c`-Flag akzeptiert jeden Befehl, der einen stdio-MCP-Server startet. Für SSE-Server `-url` verwenden.

## Server-Inspektion

Der `inspect`-Befehl analysiert einen Server auf Spec-Konformität, Metadaten-Qualität und Best Practices. Das Ergebnis ist ein Quality Score mit konkreten Hinweisen:

```bash
# Inspektion über Profil
mcp-tester inspect -p my-server

# Direktaufruf ohne Profil
mcp-tester inspect -c "./mein-mcp-server"
mcp-tester inspect -url "https://mein-server.example.com/sse"
```

Der Inspektor prüft: Tool-Beschreibungen, Input-Schema-Vollständigkeit, Resource-URI-Muster, Prompt-Argument-Definitionen und mehr.

## Profilverwaltung

Profile ermöglichen die Verwaltung mehrerer Server in einer `mcp-tester.yml`, sodass Connection-Flags nicht wiederholt werden müssen:

```bash
mcp-tester profile add local -c "./bin/mein-server"
mcp-tester profile add remote -url "https://api.example.com/mcp/sse"
mcp-tester profile list
mcp-tester profile disable local
mcp-tester profile delete local
```

## Tools, Resources & Prompts erkunden

```bash
# Tools
mcp-tester tools list -p local
mcp-tester tools call mein_tool -p local --args '{"param": "wert"}'

# Resources
mcp-tester resources list -p local
mcp-tester resources get "file:///config.yaml" -p local

# Prompts
mcp-tester prompts list -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local

# Hilfsfunktionen
mcp-tester ping -p local
mcp-tester logging debug -p local
```

## Automatisierte Test-Skripte

Für Regressionstests und CI-Integration hat MCP-Tester eine eingebaute Scripting-Engine. Skripte nutzen `.mcp`-Dateien mit einer einfachen, lesbaren Grammatik:

```mcp
# Tool aufrufen und Ergebnis speichern
call_tool add a:10 b:20
set_var result $.value

# Ergebnis prüfen
assert_equals $result "30"

# Test mit dynamischer Eingabe
input_var stadt "Stadtname eingeben:"
call_tool get_weather city:$stadt
assert_contains "temperature"
```

Skript ausführen:
```bash
mcp-tester test --script tests/mein-szenario.mcp -p local
mcp-tester test --script tests/mein-szenario.mcp -p local -v   # verbose
```

Skripte unterstützen Variablen, Typ-Konvertierung, Assertions (`assert_equals`, `assert_contains`, `assert_number`) und Pagination-Tests via Cursor.

## Transport-Unterstützung

| Transport | Flag | Einsatz |
|-----------|------|---------|
| stdio | `-c "Befehl"` | Lokale Prozesse, Binaries |
| SSE | `-url "https://..."` | Remote-Server, deployed APIs |

Beide Transporte funktionieren mit allen Befehlen: `inspect`, `tools`, `resources`, `prompts`, `test`.

## Mitgelieferter Test-Server

Der Download enthält `test-server` — eine Referenzimplementierung, die alle MCP-Features ausschöpft (Tools, Resources, Prompts, Logging, Progress, Output-Schemata). Nützlich zum Testen des eigenen MCP-Clients, nicht des mcp-tester-CLI selbst.

```bash
mcp-tester profile add everything -c "test-server"
mcp-tester inspect -p everything
```
