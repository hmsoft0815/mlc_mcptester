# MCP-Tester

Ein Command-Line Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern nach der Spezifikation vom 2025-03-26.

[English Version](README.en.md)

## Kern-Features

- **Multi-Transport**: Unterstützt lokale Prozesse (`stdio`) und Remote-Server (`sse`).
- **Full Spec Support**: Testet Tools, Resources (statisch & Templates), Subscriptions sowie Prompts.
- **Pagination Support**: Unterstützt das Durchblättern langer Listen (`list`) mittels Cursor.
- **Scripting Engine**: Automatisierte Test-Abläufe mit Variablen, Typ-Konvertierung und Assertions.
- **Server Inspector**: Analysiert Server auf Best Practices und gibt einen Quality-Score aus.
- **Raw Mode**: Umgeht SDK-Validierungen für tiefgreifendes Debugging.
- **Profile**: Einfache Verwaltung verschiedener Server in einer `mcp-tester.yml`.

---

## Der "Everything" Test-Server

Im Projekt ist ein Referenz-Server (`cmd/test-server`) enthalten, der alle Möglichkeiten des MCP-Protokolls ausschöpft (Tools, Resources, Prompts, Logging, Progress, Output-Schemata).

---

## Benutzung

### 1. Installation

**Über Go (Direkt von GitHub):**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```
*Hinweis: Stellt sicher, dass `$GOPATH/bin` (meist `~/go/bin`) in eurem `PATH` liegt.*

**Erste Schritte:**
Nach der Installation könnt ihr euren ersten Server hinzufügen und sofort testen:
```bash
# Server-Profil hinzufügen
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# Verfügbare Tools auflisten
mcp-tester tools list -p my-server
```

**Über Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manueller Build:**
```bash
task all            # Baut den Tester und Referenz-Server in den bin/ Ordner
```

### 2. Kommandos (Auszug)

#### Profilverwaltung
Verwaltet verschiedene Server-Konfigurationen direkt über die CLI:
```bash
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"
mcp-tester profile list
mcp-tester profile disable my-server
mcp-tester profile delete my-server
```

#### Server Inspektion
Analysiere einen Server auf Qualität (Metadaten, Prompts, Struktur):
```bash
# Mit Profil aus mcp-tester.yml
mcp-tester inspect --profile local

# Direktaufruf ohne Konfigurationsdatei
mcp-tester inspect -c "npx -y @modelcontextprotocol/server-everything"
```

#### Tools, Resources & Prompts
```bash
mcp-tester tools list -p local
mcp-tester resources list --cursor "NEXT_TOKEN" -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local
```

#### Test-Skripte (Automatisierung)
Führe komplexe Test-Szenarien aus:
```bash
./bin/mcp-tester test --script tests/03_variables_and_math.mcp --profile local
```

## Dokumentation

- [Das MCP-Handbuch](buch/README.md) - Eine umfassende Einführung in MCP (Deutsch).
  - [Kapitel 4: Tools – Die Hände des Modells](buch/04_tools_die_haende_des_modells.md)
  - [Kapitel 5: Resources – Das Gedächtnis des Modells](buch/05_resources_das_gedaechtnis.md)
  - [Kapitel 6: Prompts – Die Anweisungen](buch/06_prompts_die_anweisungen.md)
- [Scripting Referenz (DE)](docs/SCRIPTING.de.md) - Detaillierte Dokumentation der Test-Grammatik.
- [Scripting Reference (EN)](docs/SCRIPTING.md) - Detailed documentation of the test grammar.

---

## Lizenz
- Code: [MIT](LICENSE)
- Handbuch: [CC BY-NC-ND 4.0](buch/LICENSE.md)

---
*Copyright Michael Lechner - 2026-03-09*