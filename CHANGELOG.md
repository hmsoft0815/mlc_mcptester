# Changelog

All notable changes to mcp-tester. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versions: [SemVer](https://semver.org/).

## [Unreleased]

### Added
- `auth-check`: shows how an HTTP server is protected and which OAuth flows it offers; checks the metadata.
- `--oauth-client-credentials`: OAuth client credentials (auth extension, machine-to-machine).
- `--oauth-enterprise`, `--idp-issuer`, `--idp-client-id`, `--idp-client-secret`, `--id-token`: enterprise-managed authorization (ID-JAG).
- `apps`: lists and checks MCP Apps (`io.modelcontextprotocol/ui`): UI resources, tool linkage, CSP domains, permissions.
- Script command `verify_apps`; scripts announce the tester as a UI-capable host.
- test-server: client credentials and JWT bearer grants, test IdP (`/idp`), demo app `ui://clock`.

## [1.4.0] - 2026-09-25

### Added
- Protocol revision 2026-07-28 (go-sdk v1.8.0), fallback to older revisions.
- `http-check`: Streamable HTTP conformance (required headers, error codes, Origin, sessions).
- OAuth 2.1 authorization code flow: `--oauth`, `--oauth-auto`, `--oauth-browser`, `--oauth-client-id`, `--oauth-client-secret`, `--oauth-client-metadata-url`, `--oauth-callback-port`.
- `--bearer`, `-H/--header`; profile fields `bearer` and `headers`.
- `complete` command and script command (`completion/complete`).
- Multi round-trip requests: `elicit_response`, `sample_response`, `add_root`, `assert_elicited`, `assert_sampled`; flags `--elicit`, `--sample`, `--root`.
- Notifications via `subscriptions/listen`: `listen` command, `subscribe`, `wait_notification`.
- Tasks extension: `call --task`, `call_task`, `start_task`, `wait_task`, `get_task`, `cancel_task`, `assert_task_status`; package `pkg/mcptasks`.
- Skills extension: `skills [--verify]`, `list_skills`, `verify_skills`; package `pkg/mcpskills`.
- `inspect`: protocol lag, tool names, `title`, schema type, tool order, icon schemes, cache hints, `x-mcp-header`, extensions, skills; `--min-score`.
- `call --log-level`, `call --format json` (complete `CallToolResult`).
- `call_tool_raw` script command.
- `docs/SPEC_COVERAGE.md` with dates.

### Changed
- Scripts call tools through the SDK (per-request metadata, multi round-trip); `--raw` still bypasses it.
- `ping` uses `server/discover` and `logging` sends the level per request on 2026-07-28.
- Windows installer takes its version from `VERSION`.

### Fixed
- `inspect` reported 100/100 when listing tools, prompts or resources failed.
- `test` exited with status 0 when commands failed.
- `$.path` did not resolve inside `structuredContent`.
- `\"` escapes in double-quoted script strings.
- Only the first page of `tools/list` was read.
- `expect_error` accepted script errors (unknown command, unknown argument).
- Protocol error codes were lost over HTTP.
- ` #` and ` //` inside quotes were treated as comments; `<<` inside quotes started a heredoc.
- Unknown `name:value` arguments silently became positional arguments.
- Empty strings and empty variables disappeared as arguments.

## [1.2.0] - 2026-09-12

### Added
- Protocol revision 2025-11-25.
- Profile management (`profile`), `go install`.
- Script commands `echo`, array arguments, `assert_tool_error`.
- Copyright, website and GitHub link in `--version`.
- Windows NSIS installer, build attestation for releases.

### Fixed
- Version aligned with the release tags.
- `go vet` findings.

Older versions: [GitHub releases](https://github.com/hmsoft0815/mlc_mcptester/releases).
