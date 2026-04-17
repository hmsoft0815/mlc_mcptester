# 🔍 Detail-Dokumentation: MCP-Tester

Referenzen auf technische Detail-Dokumente innerhalb des Projekts.
Diese Dateien enthalten tiefere Informationen zu Architektur, Workflows
und Implementierungsdetails — zu umfangreich für die .mlcai/-Docs,
aber wichtig als Nachschlagewerk.

**Hinweis:** Nicht den gesamten Inhalt in den Kontext laden — gezielt
die relevante Datei lesen wenn Details zu einem bestimmten Thema nötig sind.

## Projekt-Root

| Dokument | Beschreibung | Pfad |
|------|----|----|
| [README.md](README.md) | Projektübersicht, Installation und erste Schritte | `README.md` |
| [README.en.md](README.en.md) | English version of project documentation | `README.en.md` |
| [Taskfile.yml](Taskfile.yml) | Build-Tasks, Tests, Deploy (go-task) | `Taskfile.yml` |
| [LICENSE](LICENSE) | MIT License | `LICENSE` |
| [mcp-tester.yml](mcp-tester.yml) | Beispiel-Profil-Konfiguration | `mcp-tester.yml` |

## Buch / MCP-Handbuch

Das MCP-Handbuch ist eine umfassende Einführung in das Model Context Protocol:

| Dokument | Beschreibung | Pfad |
|-----|---|----|
| [Einführung](buch/README.md) | Grundkonzept und Überblick | `buch/README.md` |
| Wie LLMs kommunizieren | Grundlagen der MCP-Kommunikation | `buch/02_wie_llms_kommunizieren.md` |
| Minimal MCP stdio | Praktischer Einstieg mit stdio | `buch/03_minimal_mcp_stdio.md` |
| Tools – Die Hände des Modells | Detail-Aufgabe zur Tool-Kompetenz | `buch/04_tools_die_haende_des_modells.md` |
| Resources – Das Gedächtnis | Ressourcen-Typen und Template-Support | `buch/05_resources_das_gedaechtnis.md` |
| Prompts – Die Anweisungen | Prompt-Strukturen und Subscriptions | `buch/06_prompts_die_anweisungen.md` |
| Kombination von Fähigkeiten | Tool-Routing und Multi-Agenten | `buch/07_die_macht_der_kombination.md` |
| Zu viele Tools, Probleme | Problembehandlung bei Tool-Explosion | `buch/08_zu_viele_tools_probleme.md` |
| Rückgabewerte (Text vs. Structured) | Datenformatierung und Strukturierte Outputs | `buch/09_rueckabewerte_text_vs_structured.md` |
| Binärdaten & Bilder | Handling von Binary-Content | `buch/10_binaerdaten_bilder.md` |
| Echtzeit-Feedback | Progress-Updates und Audio-Support | `buch/11_echtzeit_feedback_und_audio.md` |
| Grenzen und Limitierungen | Spezifikations-Beschränkungen | `buch/12_grenzen_und_limitierungen.md` |
| Artifacts Pattern | Artifact-Routing und UX-Verbesserungen | `buch/13_das_artifact_pattern.md` |
| Qualitätssicherung (inspect) | CLI-Befehl `mcp-tester inspect` | `buch/14_qualitaetssicherung_inspect.md` |
| Automatisierung CI/CD | GitHub Actions Integration | `buch/15_automatisierung_ci_cd.md` |
| Glossar | MCP- und Projekt-Begriffe | `buch/16_glossar.md` |
| SSE / HTTP2 | Protokoll-Erweiterungen (RFC) | `buch/17_anhang_sse_http2.md` |
| Notifications | Async Notifications nach RFC 07.07 | `buch/18_erweiterungen_notifications.md` |
| Tasks | Task-Support (RFC) | `buch/19_erweiterungen_tasks.md` |
| Agentischer Server | Sampling-basierte Tool-Auswahl | `buch/20_agentische_server_sampling.md` |
| Elicitation | User-Input-Einholungen | `buch/21_benutzerabfragen_elicitation.md` |
| Selbstschreiben | MCP-Spezifikationen manuell schreiben | `buch/22_selbstschreiben.md` |

## scripts/

| Dokument | Beschreibung | Pfad |
|-----|---|----|
| Install Script (Linux/macOS) | Automatisierte Installation via curl | `scripts/install.sh` |

## Tests

| Dokument | Beschreibung | Pfad |
|-----|---|----|
| [03_variables_and_math.mcp](tests/03_variables_and_math.mcp) | Beispiel-Skript: Variablen und Math | `tests/03_variables_and_math.mcp` |
| [04_resources_test.mcp](tests/04_resources_test.mcp) | Beispiel-Skript: Resource-Kontext | `tests/04_resources_test.mcp` |
| [05_basic_prompts.mcp](tests/05_basic_prompts.mcp) | Beispiel-Skript: Basic-Prompts | `tests/05_basic_prompts.mcp` |
| [06_advanced_prompts.mcp](tests/06_advanced_prompts.mcp) | Beispiel-Skript: Advanced-Prompts | `tests/06_advanced_prompts.mcp` |
| [07_resources_template.mcp](tests/07_resources_template.mcp) | Beispiel-Skript: Resource-Template | `tests/07_resources_template.mcp` |
| [11_subscriptions_basic.mcp](tests/11_subscriptions_basic.mcp) | Beispiel-Skript: Basic-Subscriptions | `tests/11_subscriptions_basic.mcp` |
| [12_subscriptions_with_cursor.mcp](tests/12_subscriptions_with_cursor.mcp) | Beispiel-Skript: Pagination | `tests/12_subscriptions_with_cursor.mcp` |
| [13_progress_events.mcp](tests/13_progress_events.mcp) | Beispiel-Skript: Progress Events | `tests/13_progress_events.mcp` |
| [15_resource_templates.mcp](tests/15_resource_templates.mcp) | Beispiel-Skript: Template-Routing | `tests/15_resource_templates.mcp` |
| [18_errors_handling.mcp](tests/18_errors_handling.mcp) | Beispiel-Skript: Error Handling | `tests/18_errors_handling.mcp` |
| [22_output_schemas.mcp](tests/22_output_schemas.mcp) | Beispiel-Skript: Output Schemas | `tests/22_output_schemas.mcp` |
| [23_resource_linking.mcp](tests/23_resource_linking.mcp) | Beispiel-Skript: Resource Linking | `tests/23_resource_linking.mcp` |
| [24_tool_reflection.mcp](tests/24_tool_reflection.mcp) | Beispiel-Skript: Tool Reflection | `tests/24_tool_reflection.mcp` |
| [34_resource_content.mcp](tests/34_resource_content.mcp) | Beispiel-Skript: Resource Content Types | `tests/34_resource_content.mcp` |

## docs/

| Dokument | Beschreibung | Pfad |
|-----|--|-|
| [Scripting Reference (DE)](docs/SCRIPTING.de.md) | Detaillierte Dokumentation der Test-Grammatik (Deutsch) | `docs/SCRIPTING.de.md` |
| [Scripting Reference (EN)](docs/SCRIPTING.md) | Detailed documentation of the test grammar (EN) | `docs/SCRIPTING.md` |

---

## 📋 Meta

- **Zuletzt aktualisiert:** 2026-04-17
- **Aktualisiert von:** qwen3.5:35b
- **Status:** Entwurf
