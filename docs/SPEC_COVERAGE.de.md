# Abdeckung der MCP-Spezifikation

**Stand: 27.09.2026** · Spezifikation **2026-07-28** (neueste stabile Revision, der Draft ist seither unverändert) · mcp-tester **1.8.1** · go-sdk **v1.8.0**

Die MCP-Spezifikation ändert sich laufend. Diese Seite hält fest, was mcp-tester zum genannten Datum prüfen kann und was nicht. Bei jeder neuen Spec-Revision oder jedem SDK-Update wird sie neu abgeglichen und das Datum angepasst.

Quellen: [Changelog 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/changelog), [Deprecated-Registry](https://modelcontextprotocol.io/specification/2026-07-28/deprecated), [go-sdk Releases](https://github.com/modelcontextprotocol/go-sdk/releases).

Die Prüfungen laufen gegen zwei unabhängige Server-Implementierungen: den Referenzserver auf dem go-sdk (`cmd/test-server`) und einen Referenzserver auf dem TypeScript-SDK v2 (`tests/interop/ts-server`, `task test-interop-ts-server`); der offizielle TypeScript-Tasks-Client läuft gegen den Go-Server (`task test-interop-ts`).

Legende: ✅ abgedeckt · ⚠️ teilweise · ❌ fehlt · — nicht anwendbar oder von außen nicht prüfbar

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
| `CacheableResult` (`ttlMs`, `cacheScope`) | ✅ | `inspect` liest die erste Seite von tools/prompts/resources/list roh bei Servern ab 2026-07-28: fehlende oder ungültige Felder (MUST) sind eine Warnung; `ttlMs` 0 und, mit Zugangsdaten, `public`-Listen sind INFO ohne Einfluss auf den Score; `--read-resources` liest die erste Resource, `public`-Inhalte mit Zugangsdaten sind eine Warnung (gemeinsame Caches dürfen sie anderen Nutzern ausliefern) |
| `serverInfo` in `_meta` der Ergebnisse | ✅ | `http-check` (discover und `tools/list`) |
| Extensions (`capabilities.extensions`) | ✅ | `inspect` zeigt sie an, JSON-Feld `extensions` |
| OpenTelemetry-`_meta` (`traceparent` …) | — | ob ein Server den Trace-Kontext weitergibt, ist von außen nicht beobachtbar |
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
| Tool-Qualität in `inspect` | ✅ | `description`, `title`, Namensregeln (1–128 Zeichen, `A-Z a-z 0-9 _ - .`), Eindeutigkeit, `inputSchema` vom Typ `object`, `outputSchema`, deterministische Reihenfolge von `tools/list`, Bonus für `readOnlyHint` (gleicht nur Qualitätsabzüge aus; MUST-Verstöße werden nach der Kappung auf 100 abgezogen) |
| Tool-Fehler (`isError`) vs. Protokollfehler | ✅ | `assert_tool_error`, `assert_error_code` |
| Icons | ✅ | `inspect` prüft die URI-Schemata (nur `https`, `data:`) bei Server, Tools, Prompts und Resources; `--check-icons` / `--download-icons` prüfen die Erreichbarkeit |
| Prompts `list` / `get` | ✅ | |
| Resources `list` / `read` / `templates` | ✅ | |
| Resource-not-found `-32602` (statt `-32002`) | ✅ | `http-check`; in Skripten per `assert_error_code -32602` |
| `subscriptions/listen`, `list_changed`, Resource-Updates | ✅ | Skriptbefehle `subscribe`, `wait_notification`; Befehl `listen`; Skripttest `14_notifications` (stdio und HTTP) |
| `completion/complete` | ✅ | Befehl `complete`, Skriptbefehl `complete`, Skripttest `12_completion` |
| Logging | ✅ | bis 2025-11-25 `setLevel`; ab 2026-07-28 pro Request über `_meta` (`call --log-level`, Skriptbefehl `logging`); deprecated seit 2026-07-28: `inspect` meldet einen 2026-07-28-Server, der die Fähigkeit noch angibt, als INFO |

## Client-Features (MRTR)

| Feature | Status | Anmerkung |
|---|---|---|
| Elicitation (Form / URL) | ✅ | beide Modi angekündigt; `assert_elicited` |
| Sampling | ✅ | `sample_response`, `assert_sampled`; deprecated seit 2026-07-28: `call` und `test` melden einen Server, der es per Multi Round-Trip anfordert, als INFO |
| Roots | ✅ | `add_root`, `--root`; deprecated seit 2026-07-28: Meldung wie bei Sampling |

## Extensions

| Feature | Status | Anmerkung |
|---|---|---|
| Tasks (`io.modelcontextprotocol/tasks`) | ✅ | Skriptbefehle `call_task`, `start_task`, `wait_task`, `get_task`, `cancel_task`, `assert_task_status`; `call --task`; Konformitätsprüfung `tasks [--tool X] [--cancel]` (Fehlercodes `-32021`/`-32602`, Form von Handle und `tasks/get`, dauerhafte Anlage, Statusübergänge, Entropie der Task-ID; `inspect` führt die Fehlercode-Prüfungen aus und zieht bei einem Verstoß 10 Punkte ab); `input_required` über `tasks/update` (Teilantworten in `pkg/mcptasks` angenommen); `Mcp-Name` = `taskId` über HTTP. Das go-sdk kennt die Extension nicht: Client über den Raw-Pfad, Server-Seite als Paket `pkg/mcptasks`. Skripttest `15_tasks`, `task test-tasks`; Interop mit dem offiziellen TypeScript-Tasks-Client (`@modelcontextprotocol/ext-tasks`) gegen den Referenzserver über stdio und Streamable HTTP (mit und ohne OAuth): `task test-interop-ts`. `notifications/tasks` (MAY): `tasks --tool` abonniert per `subscriptions/listen` mit `taskIds` und prüft Bestätigung, jede Benachrichtigung und den Endzustand gegen `tasks/get`; Abonnieren ohne Extension muss `-32021` ergeben (geprüft von `tasks` und `inspect`). Auth-Bindung: mit `--other-bearer` darf eine zweite Identität den Task weder lesen noch beantworten noch abbrechen (`-32602` erwartet, damit seine Existenz nicht durchsickert); `pkg/mcptasks` bindet Tasks an `TokenInfo.UserID` oder einen Digest des Tokens (`Store.Owner`). Das go-sdk verwirft `taskIds`, daher lehnt `pkg/mcptasks` solche Anfragen unterhalb des SDK ab (`GuardListen`, `GuardListenHandler`) und sendet selbst keine Task-Benachrichtigungen |
| Skills (`io.modelcontextprotocol/skills`) | ✅ | Befehl `skills [--verify]`, Skriptbefehle `list_skills`, `verify_skills`, Anzeige in `inspect`: Manifest (vollständig, Digests, Größen, Limits), Frontmatter nach Agent-Skills-Regeln, Name = Pfadsegment, `skills/get`, `resources/directory/read`, `-32602`. Server-Seite als Paket `pkg/mcpskills`. Skripttest `16_skills` |
| OAuth Client Credentials (ext-auth, Draft) | ✅ | `--oauth-client-credentials` über den Handler des go-sdk (Client-Secret); `auth-check` prüft `token_endpoint_auth_methods_supported`. `private_key_jwt` kann das go-sdk nicht |
| Enterprise-Managed Authorization (ext-auth, ID-JAG) | ✅ | `--oauth-enterprise` (ID-Token → Token-Exchange beim IdP → JWT-Bearer); Autorisierungsserver und Resource aus den Protected Resource Metadata; `auth-check` erkennt `authorization_grant_profiles_supported`. Den SSO-Login selbst übernimmt der Tester nicht (`--id-token`) |
| `auth-check` | ✅ | 401-Challenge, Protected Resource Metadata, Metadaten des Autorisierungsservers (issuer, PKCE S256, `iss`), angebotene Flows |
| Apps (`io.modelcontextprotocol/ui`, stabil 2026-01-26) | ⚠️ | Server-Seite: Befehl `apps`, Skriptbefehl `verify_apps` – Tool-Verknüpfung (`_meta.ui.resourceUri`, veraltetes `ui/resourceUri`), `ui://`-Schema, Resource vorhanden, MIME-Typ `text/html;profile=mcp-app`, HTML5-Dokument, `visibility`, CSP-Domains, Berechtigungen, Deklaration der Extension. Der Tester meldet sich als UI-fähiger Host an. Nicht prüfbar: die View↔Host-Kommunikation per postMessage (braucht einen Browser-Host) |

## Verlauf

| Datum | Spec | Änderung |
|---|---|---|
| 25.09.2026 | 2026-07-28 | Erster Abgleich. go-sdk v1.7.0 → v1.8.0 |
| 25.09.2026 | 2026-07-28 | Auth-Extensions (Client Credentials, Enterprise/ID-JAG), `auth-check`; MCP Apps (`apps`, `verify_apps`) |
| 25.09.2026 | 2026-07-28 | `complete`; Multi Round-Trip (Elicitation, Sampling, Roots); Notifications über `subscriptions/listen`; OAuth, Bearer, Header; `ping`/`logging` für 2026-07-28; `http-check`; `x-mcp-header` |
| 25.09.2026 | 2026-07-28 | `inspect`: Protokoll-Rückstand, Tool-Namen, `title`, Schema-Typ, Reihenfolge, Icon-Schemata, Cache-Angaben, Extensions; Pagination überall |
| 27.09.2026 | 2026-07-28 | Erneuter Abgleich: keine neuere Revision, der Draft seit 2026-07-28 nur redaktionell geändert, go-sdk v1.8.0 aktuell; Skills-Extension (SEP-2640) seit 11.09. Final, Umsetzung entspricht dem finalen Stand. Referenz-Server führt jetzt Paginierung, Cache-Hinweise, `instructions`, Tool-Annotations und Elicitation im URL-Modus vor; Skripttests 12–19 laufen in CI |
