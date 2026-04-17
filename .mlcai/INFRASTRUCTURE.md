# 🏗 Infrastructure & Deployment: mcp-tester

## Hosting & Umgebungen

| Umgebung | URL / Adresse | Anmerkung |
|----------|---------------|-----------|
| Production | GitHub Releases | `go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest` |
| Staging | — | — |
| Local Dev | `localhost` (CLI) | `./bin/mcp-tester command` oder `mcp-tester command` nach Installation |

## Deployment

- **Methode:** Git-Push → GitHub Releases (goreleaser)
- **Start-Befehl:** CLI-Kommandos `mcp-tester <command> [args]`
- **Config-Pfad:** `~/.config/mcp-tester/mcp-tester.yml` (user-global) oder `./mcp-tester.yml` (local)
- **Profil-Konfiguration:** `mcp-tester.yml` im Projekt-Root für lokale Profiler

## Abhängigkeiten (Laufzeit)

- **Go 1.24.2+** (zur编译 von Binaries)
- **npx** (für MCP-Servers wie `@playwright/mcp`, `@modelcontextprotocol/server-everything`)
- **Task** (go-task) für Build/Dev-Tasks

## Secrets / Umgebungsvariablen

| Variable | Zweck | Wo gesetzt |
|----------|-------|------------|
| `GOWORK` | Go-Workspace-Kontrolle | `Taskfile.yml` (set to `off`) |
| `GOPATH` | Go-Bin-Pfad für globale Installation | Shell environment |

## Monitoring & Alerts

- **Kein Monitoring** — CLI-Tool ohne Daemon/Service

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-17
- **Aktualisiert von:** qwen3.5:35b
- **Status:** Entwurf