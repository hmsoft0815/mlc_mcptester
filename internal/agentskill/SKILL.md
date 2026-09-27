---
name: mcp-tester
description: Test, inspect and gate MCP servers with the mcp-tester CLI — write and run .mcp test scripts, check spec conformance (inspect, http-check, auth-check, apps, skills) and produce CI results or an mcpcheck badge. Use when building, debugging or reviewing an MCP server, or when a CI job should verify one.
metadata:
  mcp-tester-version: "{{VERSION}}"
---

# mcp-tester

`mcp-tester` is a CLI **client** for MCP servers (spec 2026-07-28, falls back to
older revisions). It talks to a server over its real transport — `stdio`, SSE or
Streamable HTTP — exactly as a host would, and exits non-zero when something is
wrong. Run it from the shell; it is not an MCP server itself.

## Connecting

| Flag | Use |
|---|---|
| `-c "<command>"` | start a local server over stdio, e.g. `-c "./bin/server"` |
| `-u <url>` | remote server (Streamable HTTP; SSE if the URL ends in `/sse`) |
| `-p <profile>` | profile from `mcp-tester.yml` (command/url, bearer, headers) |
| `--bearer`, `-H "Name: value"`, `--oauth` | authorization |
| `--format json` | machine-readable output — use it whenever you parse results |

## Which command for what

| Goal | Command |
|---|---|
| Quick quality check, score 0–100, spec hints | `mcp-tester inspect -p X` |
| CI gate on quality | `mcp-tester inspect -p X --min-score 90` |
| README badge | `mcp-tester inspect -p X --badge docs/mcpcheck.svg --badge-json docs/mcpcheck.json` |
| Streamable HTTP transport rules | `mcp-tester http-check -u URL` |
| How a server is protected, OAuth flows | `mcp-tester auth-check -u URL` |
| MCP Apps / Skills extensions | `mcp-tester apps -p X`, `mcp-tester skills -p X --verify` |
| List tools with schemas | `mcp-tester list -p X` |
| One call | `mcp-tester call <tool> --args '{"k":"v"}' -p X` |
| Behaviour tests | `mcp-tester test --script tests/foo.mcp -p X` |

Protocol errors (`inspect`) and FAIL (`http-check`) are MUST violations that real
clients break on — fix them. INFO lines never affect the score. With
credentials, add `--read-resources` to `inspect`: it then warns when per-user
resource content is cacheable as `"public"`. Score hints are recommendations; a score below 100
is not automatically a bug.

Every tool result is checked against the tool's `outputSchema` the way the
official TypeScript SDK checks it: a declared schema needs matching
`structuredContent`. Results with `isError: true` are exempt.

## Writing .mcp test scripts

One command per line, `#` or `//` comments, `$var` substitution, strings in
`"…"` or `'…'`, JSON arrays/objects as quoted arguments.

```mcp
# call with named arguments (recommended over positional)
call_tool add a:2 b:3
assert_equals "Result: 5"

# read structuredContent and compare
set_var sum $.sum
assert_equals $sum "5"
assert_gt $sum 4

# arrays and objects
call_tool join_strings paths:'["a", "b"]'
assert_contains "a, b"

# expected failures: tool error vs. protocol error
expect_error call_tool error_trigger
assert_tool_error
expect_error call_tool no_such_tool
assert_error_code -32602

# elicitation / sampling / roots: prepare answers BEFORE the call
elicit_response accept '{"confirm": true}'
call_tool confirm_delete item:"report.pdf"
assert_elicited "report.pdf"

# tasks
call_task long_job seconds:1
assert_task_status completed

# notifications
subscribe mcp://time
call_tool touch_resource uri:mcp://time
wait_notification resources/updated mcp://time 2s
```

More commands: `call_tool_raw` (unchecked JSON arguments), `timeout 5000 <cmd>` (milliseconds),
`complete`, `sample_response`, `add_root`, `assert_sampled`, `start_task` /
`wait_task` / `get_task` / `cancel_task`, `list_skills` / `verify_skills`,
`verify_apps`, `assert_number`, `assert_string_length`, `echo`. Full reference:
https://github.com/hmsoft0815/mlc_mcptester/blob/main/docs/SCRIPTING.md
(update this skill with `mcp-tester agent-skill install` after upgrading).

Pitfalls:
- An unknown argument name is a script error, not a server error — check the
  tool's schema with `mcp-tester list`.
- An elicitation or sampling request without a prepared answer fails the call.
- Do not use `input_var` in CI; it waits for keyboard input.

## Running in CI

```bash
mcp-tester --format json test --script tests/smoke.mcp -p local   # stdout: summary incl. failures
mcp-tester inspect -p local --min-score 90 --badge docs/mcpcheck.svg
```

Exit code 1 on any failed command, protocol error or score below the minimum.
With `--format json`, stdout carries only the summary (with line numbers of
failures); per-command output goes to stderr. When reporting failures to a
user, quote the method, field, error code and violated rule from the output.
