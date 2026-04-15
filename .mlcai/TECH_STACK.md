# 🛠 Tech Stack & Constraints: MCP-Tester

## Kern-Versionen
- **Sprache:** Go 1.24
- **Protokoll:** Model Context Protocol (MCP) Spec 2025-03-26
- **Plattformen:** Linux (amd64, arm64), Windows (amd64), macOS (arm64, amd64)

## Bibliotheken (Erlaubt/Fixiert)
- **CLI Framework:** `github.com/spf13/cobra` (v1.10.2)
- **MCP SDK:** `github.com/modelcontextprotocol/go-sdk` (v1.4.0)
- **Konfiguration:** `gopkg.in/yaml.v3` (v3.0.1)

## Einschränkungen (Constraints)
- **Single-Binary:** Das Tool muss ohne externe Laufzeitumgebungen (außer optional npx für Node-Server) funktionieren.
- **Go 1.24:** Nutzt moderne Go-Features, erfordert mindestens Go 1.24 für den Build.
- **Cross-Platform:** Alle Pfade müssen via `path/filepath` gehandhabt werden, um Windows-Kompatibilität zu gewährleisten.

## Styling-Regeln
- **Go fmt:** Standard Go-Formatierung ist Pflicht.
- **Error Handling:** Explizites Error-Handling nach Go-Konventionen; keine versteckten Panics in der Scripting-Engine.
- **Documentation:** Alle neuen CLI-Kommandos müssen im README und via `--help` dokumentiert sein.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
