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

**Über Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manueller Build:**
```bash
task all            # Baut den Tester und Referenz-Server in den bin/ Ordner
```

### 2. Kommandos (Auszug)

#### Server Inspektion
Analysiere einen Server auf Qualität (Metadaten, Prompts, Struktur):
```bash
./bin/mcp-tester inspect --profile local
```

#### Tools, Resources & Prompts
```bash
./bin/mcp-tester tools list --profile local
./bin/mcp-tester resources list --cursor "NEXT_TOKEN" --profile local
./bin/mcp-tester resources templates --profile local
./bin/mcp-tester resources read "file:///config.json" --profile local
./bin/mcp-tester prompts get code_review --args '{"file_path": "main.go"}'
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