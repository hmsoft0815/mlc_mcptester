# MCP-Tester

Ein mächtiges Command-Line Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern.

[English Version](README.en.md)

## Kern-Features

- **Multi-Transport**: Unterstützt lokale Prozesse (`stdio`) und Remote-Server (`sse`).
- **Full Spec Support**: Testet Tools, Resources (statisch & Templates) sowie Prompts.
- **Scripting Engine**: Automatisierte Test-Abläufe mit Variablen, Typ-Konvertierung und Assertions.
- **Server Inspector**: Analysiert Server auf Best Practices und gibt einen Quality-Score aus.
- **Raw Mode**: Umgeht SDK-Validierungen für tiefgreifendes Debugging.
- **Profile**: Einfache Verwaltung verschiedener Server in einer `mcp-tester.yml`.

---

## Der "Everything" Test-Server

Im Projekt ist ein Referenz-Server (`cmd/test-server`) enthalten, der alle Möglichkeiten des MCP-Protokolls ausschöpft (Tools, Resources, Prompts, Logging, Progress, Output-Schemata).

---

## Benutzung

### 1. Installation & Build
```bash
task all            # Baut den Tester und Referenz-Server in den bin/ Ordner
```

### 2. Kommandos

#### Server Inspektion (NEU)
Analysiere einen Server auf Qualität (Metadaten, Prompts, Struktur):
```bash
./bin/mcp-tester inspect --profile local
```

#### Tools & Resources
```bash
./bin/mcp-tester list --profile local
./bin/mcp-tester call add --args '{"a": 1, "b": 2}' --profile local
./bin/mcp-tester resources read "mcp://time" --profile local
```

#### Test-Skripte (Automatisierung)
Führe komplexe Test-Szenarien aus:
```bash
./bin/mcp-tester test --script tests/03_variables_and_math.mcp --profile local
```

---

## Dokumentation

- [Das MCP-Handbuch](buch/README.md) - Eine 15-kapitlige Einführung in MCP.
- [Integrationstests](INTEGRATION_TESTS.md) - Details zum Scripting und CI/CD.
- [Prompts erklärt](PROMPTS.md) - Alles über Anweisungen und System-Kontext.

---

## Lizenz
- Code: [MIT](LICENSE)
- Handbuch: [CC BY-NC-ND 4.0](buch/LICENSE.md)

---
*Copyright Michael Lechner - 2026-02-28*
