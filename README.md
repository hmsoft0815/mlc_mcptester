# MCP-Tester

Ein mächtiges Command-Line Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern.

## Kern-Features

- **Multi-Transport**: Unterstützt lokale Prozesse (`stdio`) und Remote-Server (`sse`).
- **Full Spec Support**: Testet Tools, Resources (statisch & Templates) sowie Prompts.
- **Scripting Engine**: Automatisierte Test-Abläufe mit Typ-Konvertierung und Assertions.
- **Raw Mode**: Umgeht SDK-Validierungen, um auch nicht-standardkonforme Server-Antworten zu analysieren.
- **Profile**: Einfache Verwaltung verschiedener Server in einer `mcp-tester.yml`.

---

## Der "Everything" Test-Server

Im Projekt ist ein Referenz-Server (`test-server.go`) enthalten, der alle Möglichkeiten des MCP-Protokolls ausschöpft. Er dient als Basis für die Integrationstests.

### Tools (Funktionsaufrufe)
- `echo`: Testet einfache String-Verarbeitung.
- `add`: Validiert die **automatische Typ-Konvertierung** von Skript-Argumenten zu Integern.
- `get_image`: Testet die Übertragung von **Binärdaten** (PNG-Format).

### Resources (Datenquellen)
- **Statisch (`file:///config.json`)**: Testet das Auslesen fester JSON-Strukturen.
- **Templates (`users://{id}/profile`)**: Validiert dynamische URIs. Der Server reagiert auf Variablen innerhalb der URI.

### Prompts (Templates)
- `chat_init`: Testet die Generierung komplexer Prompts mit Argumenten und Rollenzuweisungen (`assistant`).

---

## Benutzung

### 1. Installation & Build
```bash
task build          # Baut den Tester
task build-test-server # Baut den Referenz-Server
task all            # Baut alles in den bin/ Ordner
```

### 2. Profile definieren (`mcp-tester.yml`)
```yaml
profiles:
  local:
    command: "./bin/test-server"
  sys:
    command: "../bin/system-info-server"
```

### 3. Kommandos

#### Tools auflisten & aufrufen
```bash
./bin/mcp-tester list --profile local
./bin/mcp-tester call add --profile local --args '{"a": 1, "b": 2}'
```

#### Resources & Prompts
```bash
# Resources auflisten und dynamisches Template lesen
./bin/mcp-tester resources list --profile local
./bin/mcp-tester resources read "users://42/profile" --profile local

# Prompts abrufen
./bin/mcp-tester prompts list --profile local
./bin/mcp-tester prompts get chat_init --args '{"user_name": "Alice"}' --profile local
```

#### Server Inspektion (Best Practices Check)
Analysiere einen Server auf Qualität und Vollständigkeit der Metadaten:
```bash
./bin/mcp-tester inspect --profile local
```
Dieser Befehl prüft u.a. auf fehlende Beschreibungen, fehlende Prompts (System-Kontext) und gibt einen Quality-Score aus.

#### Raw Mode (für Debugging)
Verwenden Sie `-r` oder `--raw`, wenn ein Server fehlerhafte Antworten sendet (z.B. leere `type`-Felder). Der Tester zeigt dann das rohe JSON an, anstatt mit einem SDK-Fehler abzubrechen.
```bash
./bin/mcp-tester call tool_name --profile my-server --raw
```

---

## Test-Automatisierung (Scripting)

Die Skripte erlauben es, komplexe Ketten von Aufrufen zu validieren.

**Beispiel `test.mcp`**:
```text
# Erst ein Tool aufrufen
call_tool add 10 20
# Sicherstellen, dass das Ergebnis stimmt
assert_contains "Result: 30"

# Ressourcen prüfen
# (Aktuell via CLI-Kommando im Taskfile, demnächst auch im Skript)
```

Ausführung:
```bash
./bin/mcp-tester test --script test.mcp --profile local
```

---

## Integrationstests
Das mitgelieferte `Taskfile.yml` führt automatisierte Tests gegen den `test-server` aus:
- `task test-all`: Validiert Tools, Typ-Konvertierung, Assertions, Resources und Prompts in einem Durchlauf.

## Lizenz
MIT


---
*Copyright Michael Lechner - 2026-02-28*
