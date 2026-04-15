# 📜 Decision Log (ADR)

## [2025-03-26]: Wahl von Go als Basis-Sprache
- **Kontext:** Evaluierung zwischen Python, Node.js und Go für ein plattformübergreifendes CLI-Tool.
- **Entscheidung:** Wir nutzen Go.
- **Grund:** Statische Typisierung, hervorragende Unterstützung für Single-Binary Distribution (kein Node/Python beim User nötig), native Performance für SSE-Streaming.
- **Konsequenz:** Alle Erweiterungen müssen in Go geschrieben werden; für Scripting wurde eine eigene DSL eingeführt.

## [2026-03-09]: Einführung einer Custom Scripting Engine (.mcp)
- **Kontext:** Bedarf an automatisierten Tests ohne die Komplexität von vollwertigen Programmiersprachen für den Endanwender.
- **Entscheidung:** Entwicklung einer spezifischen DSL für MCP-Tests.
- **Grund:** Einfache Syntax (`call_tool echo message:test`), direktes Mapping auf MCP-Features, einfache Integration von Assertions gegen JSON-Responses.
- **Konsequenz:** Höherer Wartungsaufwand für den Parser/Runner, aber massiv verbesserte User-Experience für Tester.

## [2026-04-10]: Integration von Profiles (mcp-tester.yml)
- **Kontext:** Nutzer müssen oft zwischen vielen verschiedenen Servern und Start-Parametern wechseln.
- **Entscheidung:** Einführung einer zentralen Konfigurationsdatei für Server-Profile.
- **Grund:** Vermeidung von langen Einzeilern in der Shell; Versionierbarkeit von Test-Setups.
- **Konsequenz:** `mcp-tester.yml` wird zum Standard-Weg für die Verwaltung von Testumgebungen.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
