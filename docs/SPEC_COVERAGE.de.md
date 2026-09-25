# Abdeckung der MCP-Spezifikation

**Stand: 25.09.2026** · Spezifikation **2026-07-28** (neueste stabile Revision, der Draft ist seither unverändert) · mcp-tester **1.4.0** · go-sdk **v1.8.0**

Die MCP-Spezifikation ändert sich laufend. Diese Seite hält fest, was mcp-tester zum genannten Datum prüfen kann und was nicht. Bei jeder neuen Spec-Revision oder jedem SDK-Update wird sie neu abgeglichen und das Datum angepasst.

Quellen: [Changelog 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/changelog), [Deprecated-Registry](https://modelcontextprotocol.io/specification/2026-07-28/deprecated), [go-sdk Releases](https://github.com/modelcontextprotocol/go-sdk/releases).

Legende: ✅ abgedeckt · ⚠️ teilweise · ❌ fehlt · — entfällt laut Spec

## Protokoll-Revisionen

| Revision | Status in mcp-tester |
|---|---|
| 2026-07-28 | ✅ wird ausgehandelt (go-sdk, `server/discover`, Metadaten pro Request) |
| 2025-11-25 … 2024-11-05 | ✅ Rückfall auf den `initialize`-Handshake für ältere Server (go-sdk) |

## Basisprotokoll

| Feature | Status | Anmerkung |
|---|---|---|
| Versionsaushandlung, Rückfall auf ältere Revisionen | ✅ | macht das go-sdk; `inspect` zeigt die ausgehandelte Version |
| Metadaten pro Request (`protocolVersion`, `clientCapabilities`, `clientInfo`) | ✅ | alle Befehle und Skripte über das go-sdk; nur `--raw` umgeht sie bewusst |
| Bewertung veralteter Revisionen | ✅ | `inspect`: 10 Punkte Abzug je Revision Rückstand (max. 30), unbekannte Version 10 |
| `server/discover` (`supportedVersions`, `instructions`, Cache-Angaben) | ✅ | `inspect` zeigt `instructions`, Capabilities und Extensions; `http-check` zeigt `supportedVersions` (über HTTP) |
| `resultType` / Multi Round-Trip (`InputRequiredResult`) | ✅ | über das go-sdk; Antworten per Skript (`elicit_response`, `sample_response`, `add_root`) oder `--elicit` / `--sample` / `--root`; Skripttest `13_input_requests` |
| `CacheableResult` (`ttlMs`, `cacheScope`) | ✅ | `inspect` prüft die erste Seite von tools/prompts/resources/list bei Servern ab 2026-07-28 |
| `serverInfo` in `_meta` der Ergebnisse | ❌ | nicht geprüft |
| Extensions (`capabilities.extensions`) | ✅ | `inspect` zeigt sie an, JSON-Feld `extensions` |
| OpenTelemetry-`_meta` (`traceparent` …) | ❌ | |
| Fortschritt (`progressToken`) | ✅ | wird angezeigt, Skripttest `05_progress` |
| Abbruch (`notifications/cancelled`) | ✅ | Skripttest `06_cancellation` |
| Pagination | ✅ | `inspect`, `list` und die Skript-Engine folgen `nextCursor` über alle Seiten; `prompts list` / `resources list` blättern mit `--cursor` |
| Fehlercodes | ✅ | `assert_error_code` prüft beliebige Codes, auch -32020…-32022 |
| `ping` | ✅ | in 2026-07-28 entfernt: dort gilt `server/discover` als Lebenszeichen, sonst `ping` |

## Transporte

| Feature | Status | Anmerkung |
|---|---|---|
| stdio | ✅ | |
| Streamable HTTP | ✅ | über das go-sdk |
| HTTP+SSE (Legacy) | ✅ | in 2026-07-28 als deprecated eingestuft |
| Header `MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, Base64-Werte | ✅ | `http-check`: fehlend und abweichend → 400/`-32020`, Base64 muss dekodiert werden |
| `x-mcp-header` / `Mcp-Param-*` | ✅ | `inspect` prüft die Annotationen (Token, eindeutig, Typ, nur über `properties` erreichbar); `http-check` prüft die serverseitige Validierung |
| Origin-Prüfung (403), 404 für unbekannte Methode, 405 für GET/DELETE, keine Sessions | ✅ | `http-check` |
| Fehler bei unbekannter Version (400/`-32022`), fehlende `_meta` (400/`-32602`) | ✅ | `http-check` |

## Autorisierung

| Feature | Status | Anmerkung |
|---|---|---|
| OAuth 2.1 / PKCE, Protected Resource Metadata, `iss`-Prüfung | ✅ | `--oauth` über den Handler des go-sdk; `--oauth-auto` ohne Browser; geprüft gegen `test-server -auth` |
| Statische Zugangsdaten | ✅ | `--bearer`, `-H/--header`, Profilfelder `bearer` / `headers` |
| Client ID Metadata Documents, Dynamic Client Registration | ✅ | `--oauth-client-metadata-url`, `--oauth-client-id`, sonst dynamische Registrierung (deprecated seit 2026-07-28) |

## Server-Features

| Feature | Status | Anmerkung |
|---|---|---|
| `tools/list`, `tools/call` | ✅ | `list`, `call`, Skripte |
| `outputSchema` ↔ `structuredContent` | ✅ | `call` und Skripte prüfen wie ein strikter Client |
| Tool-Qualität in `inspect` | ✅ | `description`, `title`, Namensregeln (1–128 Zeichen, `A-Z a-z 0-9 _ - .`), Eindeutigkeit, `inputSchema` vom Typ `object`, `outputSchema`, deterministische Reihenfolge von `tools/list`, Bonus für `readOnlyHint` |
| Tool-Fehler (`isError`) vs. Protokollfehler | ✅ | `assert_tool_error`, `assert_error_code` |
| Icons | ✅ | `inspect` prüft die URI-Schemata (nur `https`, `data:`) bei Server, Tools, Prompts und Resources; `--check-icons` / `--download-icons` prüfen die Erreichbarkeit |
| Prompts `list` / `get` | ✅ | |
| Resources `list` / `read` / `templates` | ✅ | |
| Resource-not-found `-32602` (statt `-32002`) | ⚠️ | per `assert_error_code` prüfbar, `inspect` prüft es nicht |
| `subscriptions/listen`, `list_changed`, Resource-Updates | ✅ | Skriptbefehle `subscribe`, `wait_notification`; Befehl `listen`; Skripttest `14_notifications` (stdio und HTTP) |
| `completion/complete` | ✅ | Befehl `complete`, Skriptbefehl `complete`, Skripttest `12_completion` |
| Logging | ✅ | bis 2025-11-25 `setLevel`; ab 2026-07-28 pro Request über `_meta` (`call --log-level`, Skriptbefehl `logging`); deprecated seit 2026-07-28 |

## Client-Features (MRTR)

| Feature | Status | Anmerkung |
|---|---|---|
| Elicitation (Form / URL) | ✅ | beide Modi angekündigt; `assert_elicited` |
| Sampling | ✅ | `sample_response`, `assert_sampled`; deprecated seit 2026-07-28 |
| Roots | ✅ | `add_root`, `--root`; deprecated seit 2026-07-28 |

## Extensions

| Feature | Status | Anmerkung |
|---|---|---|
| Tasks (`io.modelcontextprotocol/tasks`) | ❌ | |
| Apps, Skills, Auth-Extensions | ❌ | |

## Verlauf

| Datum | Spec | Änderung |
|---|---|---|
| 25.09.2026 | 2026-07-28 | Erster Abgleich. go-sdk v1.7.0 → v1.8.0 |
| 25.09.2026 | 2026-07-28 | `inspect`: Protokoll-Rückstand, Tool-Namen, `title`, Schema-Typ, Reihenfolge, Icon-Schemata, Cache-Angaben, Extensions; Pagination überall |
