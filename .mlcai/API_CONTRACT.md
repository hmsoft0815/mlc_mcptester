# 🔌 API Contract: MCP-Tester

## 1. CLI-Kommandos (vollständig)

mcp-tester agiert als **MCP-Client** und bietet folgende Top-Level-Commands:

### `list` — Tools auflisten
Verbindet sich mit dem Server und listet alle verfügbaren Tools mit Schemas.
```
mcp-tester list --command "./bin/test-server"
mcp-tester list --profile local
mcp-tester list --url http://localhost:8081/sse
```

### `call` — Tool aufrufen
Ruft ein einzelnes Tool auf dem Server auf und zeigt das Ergebnis.
```
mcp-tester call echo --command "./bin/test-server" --args '{"message":"hello"}'
mcp-tester call add --profile local --raw   # Raw Mode: bypasses SDK validation
```
**`--raw` Flag:** Umgeht die SDK-Validierung und sendet/empfängt rohe JSON-RPC-Nachrichten — nützlich für Deep-Level-Debugging und non-konformer Server-Implementierungen.

### `test` — Scripting Engine
Führt eine `.mcp`-Scriptdatei aus (automatisierte Testszenarien).
```
mcp-tester test --profile local --script tests/01_simple.mcp
mcp-tester test --command "./bin/test-server" --script my_test.mcp -v
```

### `inspect` — Server Inspector
Analysiert den Server und berechnet einen **Quality Score** (0–100).
Prüft: Tool-Beschreibungen, Prompt-Definitionen, Resource-Verfügbarkeit, Protokoll-Konformität.
Gibt Recommendations aus und liefert einen JSON-Report (`--format json`).
```
mcp-tester inspect --profile local
mcp-tester inspect --url http://localhost:8081/sse --format json
```
Report-Felder: `serverName`, `serverVersion`, `protocolVersion`, `score`, `recommendations`, `toolsFound`, `promptsFound`, `resourcesFound`

### `ping` — Verbindungstest
Sendet einen MCP Ping-Request und misst die Antwortzeit.
```
mcp-tester ping --profile local
```

### `logging` — Log-Level setzen
Setzt den Log-Level auf dem Server via MCP `setLevel`.
```
mcp-tester logging debug --profile local
mcp-tester logging info --command "./bin/test-server"
```

### `resources` — Resources verwalten
Sub-Commands: `list`, `templates list`, `read`
```
mcp-tester resources list --profile local
mcp-tester resources templates list --profile local
mcp-tester resources read res://my-resource --profile local
```
Paginierung via `--cursor` Flag.

### `prompts` — Prompts verwalten
Sub-Commands: `list`, `get`
```
mcp-tester prompts list --profile local
mcp-tester prompts get my-prompt --profile local
```
Paginierung via `--cursor` Flag.

### `profile` — Server-Profile verwalten
Verwaltet Einträge in `mcp-tester.yml` (add, list, remove).
```
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"
mcp-tester profile list
```

---

## 2. Transport-Flags (global)

| Flag | Beschreibung |
|------|-------------|
| `--command` / `-c` | Startet lokalen Prozess (stdio-Transport) |
| `--url` / `-u` | Verbindet per SSE (HTTP-Transport) |
| `--profile` / `-p` | Nutzt Profil aus `mcp-tester.yml` |
| `--verbose` / `-v` | Verbose-Output |
| `--format` | `text` (default) oder `json` |

---

## 3. Scripting DSL (`.mcp`)

Dateiendung: `.mcp` — eine Anweisung pro Zeile, `//` für Kommentare.

| Command | Beispiel | Beschreibung |
|---------|---------|--------------|
| `call_tool` | `call_tool echo message:hello` | Tool aufrufen (named oder positional args) |
| `set_var` | `set_var myvar structuredContent.result` | Variable aus JSON-Pfad setzen |
| `assert_contains` | `assert_contains "Result: 30"` | Prüft ob Response-Text vorkommt |
| `assert_equals` | `assert_equals $myvar 30` | Exakter Wert-Vergleich |
| `assert_gt` | `assert_gt $myvar 4` | Numerisch größer als |
| `assert_number` | `assert_number $myvar` | Prüft ob Variable eine Zahl ist |
| `expect_error` | `expect_error` | Nächster call_tool muss mit Fehler antworten |

Variablen mit `$` Prefix, JSON-Pfad-Zugriff mit Punktnotation.

---

## 4. Referenz-Server (`cmd/test-server`)

Beinhaltet einen vollständigen Referenz-Server (alle MCP-Features):
- **Transport:** SSE (`--addr :8081` default) und stdio
- **Endpoint:** `http://localhost:8081/sse`
- **Tools:** `echo`, `add`, `long_running` (mit Progress/Cancellation)
- **Resources:** statische + Templates, Subscriptions
- **Prompts:** Beispiel-Prompts

---

## 5. Profil-Konfiguration (`mcp-tester.yml`)

```yaml
profiles:
  local:
    command: ./bin/test-server
  my-remote:
    url: http://remote-host:8081/sse
```

---

## 6. Installation

```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```
Hinweis: GitHub-Account `hmsoft0815` — nicht `mlcgo.eu`-Namespace.

---

## 7. Globale Konventionen
- **Protokoll-Version:** MCP Spec **2025-11-25**
- **JSON-RPC 2.0:** Alle MCP-Nachrichten
- **Paginierung:** cursor-basiert für `list`-Commands (Tools, Resources, Prompts)

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
