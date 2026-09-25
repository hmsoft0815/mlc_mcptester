# MCP Specification Coverage

**As of 2026-09-25** · specification **2026-07-28** (latest stable revision; the draft is unchanged since) · mcp-tester **1.4.0** · go-sdk **v1.8.0**

The MCP specification keeps changing. This page records what mcp-tester can and cannot check as of the date above. It is re-checked, and the date updated, with every new spec revision or SDK update.

Sources: [changelog 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/changelog), [deprecated registry](https://modelcontextprotocol.io/specification/2026-07-28/deprecated), [go-sdk releases](https://github.com/modelcontextprotocol/go-sdk/releases).

Legend: ✅ covered · ⚠️ partial · ❌ missing · — not applicable per spec

## Protocol revisions

| Revision | Status in mcp-tester |
|---|---|
| 2026-07-28 | ✅ negotiated (go-sdk, `server/discover`, per-request metadata) |
| 2025-11-25 … 2024-11-05 | ✅ falls back to the `initialize` handshake for older servers (go-sdk) |

## Base protocol

| Feature | Status | Notes |
|---|---|---|
| Version negotiation, fallback to older revisions | ✅ | done by the go-sdk; `inspect` shows the negotiated version |
| Per-request metadata (`protocolVersion`, `clientCapabilities`, `clientInfo`) | ✅ | all commands and scripts via the go-sdk; only `--raw` deliberately bypasses it |
| Rating outdated revisions | ✅ | `inspect`: 10 points per revision behind (max. 30), unknown version 10 |
| `server/discover` (`supportedVersions`, `instructions`, cache hints) | ✅ | `inspect` shows `instructions`, capabilities and extensions; `http-check` shows `supportedVersions` (over HTTP) |
| `resultType` / multi round-trip (`InputRequiredResult`) | ✅ | via the go-sdk; answers from scripts (`elicit_response`, `sample_response`, `add_root`) or `--elicit` / `--sample` / `--root`; script test `13_input_requests` |
| `CacheableResult` (`ttlMs`, `cacheScope`) | ✅ | `inspect` checks the first page of tools/prompts/resources/list for servers on 2026-07-28 or later |
| `serverInfo` in result `_meta` | ✅ | `http-check` (discover and `tools/list`) |
| Extensions (`capabilities.extensions`) | ✅ | shown by `inspect`, JSON field `extensions` |
| OpenTelemetry `_meta` (`traceparent` …) | — | whether a server propagates the trace context is not observable from outside |
| Progress (`progressToken`) | ✅ | displayed, script test `05_progress` |
| Cancellation (`notifications/cancelled`) | ✅ | script test `06_cancellation` |
| Pagination | ✅ | `inspect`, `list` and the script engine follow `nextCursor` across all pages; `prompts list` / `resources list` page with `--cursor` |
| Error codes | ✅ | `assert_error_code` checks any code, including -32020…-32022 |
| `ping` | ✅ | removed in 2026-07-28: there `server/discover` counts as liveness, otherwise `ping` |

## Transports

| Feature | Status | Notes |
|---|---|---|
| stdio | ✅ | |
| Streamable HTTP | ✅ | via the go-sdk |
| HTTP+SSE (legacy) | ✅ | classified as deprecated in 2026-07-28 |
| `MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name` headers, Base64 values | ✅ | `http-check`: missing and mismatching → 400/`-32020`, Base64 must be decoded |
| `x-mcp-header` / `Mcp-Param-*` | ✅ | `inspect` checks the annotations (token, unique, type, reachable only via `properties`); `http-check` checks server-side validation |
| Origin check (403), 404 for unknown method, 405 for GET/DELETE, no sessions | ✅ | `http-check` |
| Unknown version (400/`-32022`), missing `_meta` (400/`-32602`) | ✅ | `http-check` |

## Authorization

| Feature | Status | Notes |
|---|---|---|
| OAuth 2.1 / PKCE, Protected Resource Metadata, `iss` validation | ✅ | `--oauth` via the go-sdk handler; `--oauth-auto` without a browser; checked against `test-server -auth` |
| Static credentials | ✅ | `--bearer`, `-H/--header`, profile fields `bearer` / `headers` |
| Client ID Metadata Documents, Dynamic Client Registration | ✅ | `--oauth-client-metadata-url`, `--oauth-client-id`, otherwise dynamic registration (deprecated since 2026-07-28) |

## Server features

| Feature | Status | Notes |
|---|---|---|
| `tools/list`, `tools/call` | ✅ | `list`, `call`, scripts |
| `outputSchema` ↔ `structuredContent` | ✅ | `call` and scripts check like a strict client |
| Tool quality in `inspect` | ✅ | `description`, `title`, naming rules (1–128 chars, `A-Z a-z 0-9 _ - .`), uniqueness, `inputSchema` of type `object`, `outputSchema`, deterministic `tools/list` order, bonus for `readOnlyHint` |
| Tool errors (`isError`) vs. protocol errors | ✅ | `assert_tool_error`, `assert_error_code` |
| Icons | ✅ | `inspect` checks URI schemes (`https`, `data:` only) on server, tools, prompts and resources; `--check-icons` / `--download-icons` check reachability |
| Prompts `list` / `get` | ✅ | |
| Resources `list` / `read` / `templates` | ✅ | |
| Resource not found `-32602` (was `-32002`) | ⚠️ | checkable via `assert_error_code`, not checked by `inspect` |
| `subscriptions/listen`, `list_changed`, resource updates | ✅ | script commands `subscribe`, `wait_notification`; `listen` command; script test `14_notifications` (stdio and HTTP) |
| `completion/complete` | ✅ | `complete` command and script command, script test `12_completion` |
| Logging | ✅ | up to 2025-11-25 `setLevel`; from 2026-07-28 per request via `_meta` (`call --log-level`, script command `logging`); deprecated since 2026-07-28 |

## Client features (MRTR)

| Feature | Status | Notes |
|---|---|---|
| Elicitation (form / URL) | ✅ | both modes declared; `assert_elicited` |
| Sampling | ✅ | `sample_response`, `assert_sampled`; deprecated since 2026-07-28 |
| Roots | ✅ | `add_root`, `--root`; deprecated since 2026-07-28 |

## Extensions

| Feature | Status | Notes |
|---|---|---|
| Tasks (`io.modelcontextprotocol/tasks`) | ❌ | |
| Apps, Skills, auth extensions | ❌ | |

## History

| Date | Spec | Change |
|---|---|---|
| 2026-09-25 | 2026-07-28 | First review. go-sdk v1.7.0 → v1.8.0 |
| 2026-09-25 | 2026-07-28 | `complete`; multi round-trip (elicitation, sampling, roots); notifications via `subscriptions/listen`; OAuth, bearer, headers; `ping`/`logging` for 2026-07-28; `http-check`; `x-mcp-header` |
| 2026-09-25 | 2026-07-28 | `inspect`: protocol lag, tool names, `title`, schema type, order, icon schemes, cache hints, extensions; pagination everywhere |
