# 📚 Nutzer-Dokumentation: MCP-Tester

Übersicht der Dokumentation für Endnutzer und Entwickler die das
Projekt verwenden (nicht entwickeln) wollen.

## Für Endnutzer

| Dokument | Beschreibung | Pfad |
|------|----|----|
| [Erste Schritte](README.md) | Installation, Kommando-Übersicht | `README.md` |
| [Scripting Reference DE](docs/SCRIPTING.de.md) | Test-Grammatik und Assertions | `docs/SCRIPTING.de.md` |
| [Scripting Reference EN](docs/SCRIPTING.md) | Test-Grammar and assertions | `docs/SCRIPTING.md` |
| [MCP-Handbuch](buch/README.md) | Umfassende MCP-Einführung (13 Kapitel) | `buch/README.md` |
| FAQ / Known Issues | - | [GitHub Issues](https://github.com/hmsoft0815/mlc_mcptester/issues) |

## Für Entwickler

| Dokument | Beschreibung | Pfad |
|------|----|----|
| [Build-Anleitung](README.md) | `go install`, `curl`-Installation, manueller Build | `README.md` |
| [Task-Referenz](Taskfile.yml) | Verfügbare Build-Tasks (`task --list`) | `Taskfile.yml` |
| [MLC Doc Hub Docs](.mlcai/INTEGRATION.md) | Integration, Deployment, API-Contract | `.mlcai/INTEGRATION.md` |
| [Tech Stack](.mlcai/TECH_STACK.md) | Abhängigkeiten, Versionen | `.mlcai/TECH_STACK.md` |
| [Architektur-Entscheidungen](.mlcai/DECISION_LOG.md) | ADRs, Design-Entscheidungen | `.mlcai/DECISION_LOG.md` |

## Plattform-Support

| Plattform | Format | Hinweise |
|-----------|-|----|
| Linux | Binary, Go Module (1.24.2+) | `linux-amd64`, `linux-arm64` durch GoReleaser |
| macOS | Binary, Go Module (1.24.2+) | `darwin-amd64`, `darwin-arm64` durch GoReleaser |
| Windows | Binary, Go Module (1.24.2+) | `windows-amd64` durch GoReleaser |

## Weitere Ressourcen

- **Dokumentation:** [MLC Doc Hub](https://github.com/hmsoft0815/mlc_mcptester/tree/main/.mlcai) — alle .mlcai/-Docs sind via Doc Hub abrufbar.
- **Bug Reports:** [GitHub Issues](https://github.com/hmsoft0815/mlc_mcptester/issues)
- **Source Code:** [GitHub Repository](https://github.com/hmsoft0815/mlc_mcptester)
- **CI/CD Status:** [GitHub Actions](https://github.com/hmsoft0815/mlc_mcptester/actions)

Technische Details, API-Referenz und Architektur sind in den
.mlcai/-Docs im Doc Hub dokumentiert — hier nicht wiederholen.

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-17
- **Aktualisiert von:** qwen3.5:35b
- **Status:** Entwurf
