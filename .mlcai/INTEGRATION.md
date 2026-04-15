# 🤖 AI & Integration Context: MCP-Tester

## 1. Identität & Zweck
- **Kernaufgabe:** Command-Line Tool zum Testen, Debuggen und Validieren von Model Context Protocol (MCP) Servern nach der Spezifikation vom 2025-03-26.
- **Technischer Stack:** Go 1.24, Model Context Protocol Go SDK, Cobra CLI.
- **Hoster/Infrastruktur:** Lokale Ausführung (Single-Binary), unterstützt Linux, macOS und Windows.

## 2. Die "Nachbarschaft" (System-Kontext)
- **Upstream (Wovon hänge ich ab?):**
  - [Model Context Protocol (MCP)](https://modelcontextprotocol.io) -> Offizielle Spezifikation für die Kommunikation zwischen LLMs und Tools/Resources.
- **Downstream (Wer nutzt mich?):**
  - Entwickler von MCP-Servern (z.B. `hub-osm`, `system-info-server`, `wollmilchsau`) zur Validierung ihrer Implementierungen.
  - CI/CD-Pipelines zur automatisierten Prüfung von MCP-Schnittstellen.

## 3. Schnittstellen-Vertrag
- **Primäre API:** Implementiert einen MCP-Client, der über `stdio` (lokale Prozesse) oder `sse` (HTTP/SSE) mit Servern kommuniziert.
- **Scripting Engine:** Eigene DSL (`.mcp`) für automatisierte Testabläufe mit Assertions und Variablen.
- **Server Inspector:** `inspect`-Command mit Quality Score (0–100) und Recommendations.
- **Raw Mode:** `--raw` Flag beim `call`-Command umgeht SDK-Validierung für Deep-Level-Debugging.
- **Referenz-Server:** Beinhaltet einen `test-server` (stdio + SSE), der alle MCP-Features exponiert (Standard-Port `:8081`).

## 4. Leitplanken & Regeln
- **Naming:** Go-Standard-Konventionen; CLI-Kommandos folgen dem Cobra-Pattern.
- **Testing:** Umfassende Test-Suite in `tests/` für die Scripting Engine und Kern-Funktionalitäten.
- **Taskfile First:** Alle Build- und Test-Aufgaben werden über `Taskfile.yml` gesteuert.

## 5. Aktueller Fokus (Status)
- **Status:** Beta (v0.2.3).
- **MCP-Spec:** 2025-11-25 (aktuelle Implementierung; ältere Spec-Versionen könnten Inkompatibilitäten aufweisen).
- **Nächste Schritte:** Erweiterung der Scripting-Engine um komplexere Datenstrukturen (Maps, Arrays in Variablen).

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
