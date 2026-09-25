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
| Rating outdated revisions | ✅ | `inspect`: 10 points per revision behind (max. 30), unknown version 10 |
| `server/discover` (`supportedVersions`, `instructions`, cache hints) | ⚠️ | `inspect` shows `instructions`, capabilities and extensions; the go-sdk does not expose `supportedVersions` or discover's cache hints |
| `resultType` / multi round-trip (`InputRequiredResult`) | ❌ | the tester declares no client capabilities and answers no `inputRequests` |
| `CacheableResult` (`ttlMs`, `cacheScope`) | ✅ | `inspect` checks the first page of tools/prompts/resources/list for servers on 2026-07-28 or later |
| `serverInfo` in result `_meta` | ❌ | not checked |
| Extensions (`capabilities.extensions`) | ✅ | shown by `inspect`, JSON field `extensions` |
| OpenTelemetry `_meta` (`traceparent` …) | ❌ | |
| Progress (`progressToken`) | ✅ | displayed, script test `05_progress` |
| Cancellation (`notifications/cancelled`) | ✅ | script test `06_cancellation` |
| Pagination | ✅ | `inspect`, `list` and the script engine follow `nextCursor` across all pages; `prompts list` / `resources list` page with `--cursor` |
| Error codes | ✅ | `assert_error_code` checks any code, including -32020…-32022 |
| `ping` | ⚠️ | removed in 2026-07-28; the command stays for older servers |

## Transports

| Feature | Status | Notes |
|---|---|---|
| stdio | ✅ | |
| Streamable HTTP | ✅ | via the go-sdk |
| HTTP+SSE (legacy) | ✅ | classified as deprecated in 2026-07-28 |
| `Mcp-Method`, `Mcp-Name`, `x-mcp-header` headers | ❌ | not checked |
| Origin check (403), 404 for unknown method | ❌ | |

## Authorization

| Feature | Status | Notes |
|---|---|---|
| OAuth 2.1 / PKCE, Protected Resource Metadata, `iss` validation | ❌ | no token or header support; protected servers cannot be tested |
| Client ID Metadata Documents, Dynamic Client Registration | ❌ | |

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
| `subscriptions/listen`, `list_changed`, resource updates | ❌ | |
| `completion/complete` | ✅ | `complete` command and script command, script test `12_completion` |
| Logging | ⚠️ | `logging` uses `setLevel`; deprecated in 2026-07-28, where the level is set per request via `_meta` |

## Client features (MRTR)

| Feature | Status | Notes |
|---|---|---|
| Elicitation (form / URL) | ❌ | |
| Sampling | ❌ | deprecated since 2026-07-28 |
| Roots | ❌ | deprecated since 2026-07-28 |

## Extensions

| Feature | Status | Notes |
|---|---|---|
| Tasks (`io.modelcontextprotocol/tasks`) | ❌ | |
| Apps, Skills, auth extensions | ❌ | |

## History

| Date | Spec | Change |
|---|---|---|
| 2026-09-25 | 2026-07-28 | First review. go-sdk v1.7.0 → v1.8.0 |
| 2026-09-25 | 2026-07-28 | `inspect`: protocol lag, tool names, `title`, schema type, order, icon schemes, cache hints, extensions; pagination everywhere |
