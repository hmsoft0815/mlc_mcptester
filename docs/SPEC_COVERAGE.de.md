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
| Bewertung veralteter Revisionen | ❌ | ein Server nur mit 2024-11-05 bekommt trotzdem 100/100 |
| `server/discover` (`supportedVersions`, `instructions`, Cache-Angaben) | ❌ | wird nicht angezeigt oder geprüft |
| `resultType` / Multi Round-Trip (`InputRequiredResult`) | ❌ | der Tester gibt keine Client-Fähigkeiten an und beantwortet keine `inputRequests` |
| `CacheableResult` (`ttlMs`, `cacheScope`) | ❌ | nicht geprüft |
| `serverInfo` in `_meta` der Ergebnisse | ❌ | nicht geprüft |
| Extensions (`capabilities.extensions`) | ❌ | nicht angezeigt |
| OpenTelemetry-`_meta` (`traceparent` …) | ❌ | |
| Fortschritt (`progressToken`) | ✅ | wird angezeigt, Skripttest `05_progress` |
| Abbruch (`notifications/cancelled`) | ✅ | Skripttest `06_cancellation` |
| Pagination | ⚠️ | `prompts list` / `resources list` mit `--cursor`; `inspect`, `list` und die Skript-Engine lesen nur die **erste Seite** von `tools/list` |
| Fehlercodes | ✅ | `assert_error_code` prüft beliebige Codes, auch -32020…-32022 |
| `ping` | ⚠️ | in 2026-07-28 entfernt; der Befehl bleibt für ältere Server |

## Transporte

| Feature | Status | Anmerkung |
|---|---|---|
| stdio | ✅ | |
| Streamable HTTP | ✅ | über das go-sdk |
| HTTP+SSE (Legacy) | ✅ | in 2026-07-28 als deprecated eingestuft |
| Header `Mcp-Method`, `Mcp-Name`, `x-mcp-header` | ❌ | werden nicht geprüft |
| Origin-Prüfung (403), 404 für unbekannte Methode | ❌ | |

## Autorisierung

| Feature | Status | Anmerkung |
|---|---|---|
| OAuth 2.1 / PKCE, Protected Resource Metadata, `iss`-Prüfung | ❌ | keine Unterstützung für Tokens oder Header; geschützte Server sind nicht testbar |
| Client ID Metadata Documents, Dynamic Client Registration | ❌ | |

## Server-Features

| Feature | Status | Anmerkung |
|---|---|---|
| `tools/list`, `tools/call` | ✅ | `list`, `call`, Skripte |
| `outputSchema` ↔ `structuredContent` | ✅ | `call` und Skripte prüfen wie ein strikter Client |
| Tool-Qualität in `inspect` | ⚠️ | geprüft: `description`, `inputSchema`, `outputSchema`, `readOnlyHint`. Nicht geprüft: `title`, Namensregeln (1–128 Zeichen, Zeichensatz), weitere Annotations, deterministische Reihenfolge, `inputSchema` vom Typ `object` |
| Tool-Fehler (`isError`) vs. Protokollfehler | ✅ | `assert_tool_error`, `assert_error_code` |
| Icons | ⚠️ | `--check-icons` / `--download-icons`; die erlaubten URI-Schemata (nur `https`, `data:`) werden nicht geprüft |
| Prompts `list` / `get` | ✅ | |
| Resources `list` / `read` / `templates` | ✅ | |
| Resource-not-found `-32602` (statt `-32002`) | ⚠️ | per `assert_error_code` prüfbar, `inspect` prüft es nicht |
| `subscriptions/listen`, `list_changed`, Resource-Updates | ❌ | |
| `completion/complete` | ❌ | kein Befehl |
| Logging | ⚠️ | `logging` nutzt `setLevel`; in 2026-07-28 deprecated, die Stufe wird dort pro Request über `_meta` gesetzt |

## Client-Features (MRTR)

| Feature | Status | Anmerkung |
|---|---|---|
| Elicitation (Form / URL) | ❌ | |
| Sampling | ❌ | deprecated seit 2026-07-28 |
| Roots | ❌ | deprecated seit 2026-07-28 |

## Extensions

| Feature | Status | Anmerkung |
|---|---|---|
| Tasks (`io.modelcontextprotocol/tasks`) | ❌ | |
| Apps, Skills, Auth-Extensions | ❌ | |

## Verlauf

| Datum | Spec | Änderung |
|---|---|---|
| 25.09.2026 | 2026-07-28 | Erster Abgleich. go-sdk v1.7.0 → v1.8.0 |
