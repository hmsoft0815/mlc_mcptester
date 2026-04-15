# 🛠 Tech Stack & Constraints: MCP-Tester

## Kern-Versionen
- **Sprache:** Go 1.24 (go.mod: `go 1.24.2`)
- **Protokoll:** Model Context Protocol (MCP) Spec **2025-11-25**
- **Plattformen:** Linux (amd64, arm64), Windows (amd64), macOS (arm64, amd64)
- **Version:** 0.2.3 (`internal/version/version.go`)

## Bibliotheken (Erlaubt/Fixiert)
- **CLI Framework:** `github.com/spf13/cobra` (v1.10.2)
- **MCP SDK:** `github.com/modelcontextprotocol/go-sdk` (v1.4.0)
- **Konfiguration:** `gopkg.in/yaml.v3` (v3.0.1)
- **JSON/Encoding:** `github.com/segmentio/encoding` (v0.5.3, transitiv)
- **JSON Schema:** `github.com/google/jsonschema-go` (v0.4.2, transitiv)

## Build-Tooling
- **Task Runner:** `Taskfile.yml` — alle Build- und Test-Aufgaben via `task`
- **Release:** `goreleaser` (`.goreleaser.yaml`) — Cross-Platform Release-Builds
- **Windows Installer:** `makensis` (NSIS) — `scripts/windows-installer.nsi`
- **Installer-Script (Unix):** `scripts/install.sh`
- **QA:** `staticcheck` + `gocognit` (optionale Tasks, kein Pflicht-Gate)

## Einschränkungen (Constraints)
- **Single-Binary:** Kein externe Laufzeitumgebungen nötig (außer optional `npx` für Node-basierte MCP-Server).
- **GOWORK=off:** Alle Build- und Test-Commands setzen `GOWORK=off` explizit, da das Projekt in einem Go-Workspace-Kontext liegt.
- **Cross-Platform:** Alle Pfade via `path/filepath`; Windows-Kompatibilität ist Pflicht.
- **Go-Install-Pfad:** `github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester` (GitHub-Account `hmsoft0815`, nicht `mlcgo`).

## Styling-Regeln
- **Go fmt:** Standard Go-Formatierung ist Pflicht (`task format`).
- **Error Handling:** Explizites Error-Handling nach Go-Konventionen; keine versteckten Panics in der Scripting-Engine.
- **Documentation:** Alle neuen CLI-Kommandos müssen im README und via `--help` dokumentiert sein.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-15
- **Aktualisiert von:** Gemini CLI (v0.2.3)
- **Status:** Aktuell
