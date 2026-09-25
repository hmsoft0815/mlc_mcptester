# MCP-Tester

<img src="docs/assets/hero.jpg" alt="MCP-Tester — illustration" width="820">


> **[mlcgo.eu](https://mlcgo.eu)** — tools, libraries and manuals · [Product page](https://mlcgo.eu/products/mlc-tester/)


A command-line tool for testing, debugging, and validating Model Context Protocol (MCP) servers based on the latest 2026-07-28 specification.

[Deutsche Version](README.md)

## Why MCP-Tester?

New hosts, harnesses and models appear almost daily, and the MCP specification keeps evolving. A server that does not follow the specification closely then "suddenly" stops working well: calls fail with stricter clients, error messages do not help the model, or the model burns tokens on retries. `mcp-tester` helps you stay current quickly and with little effort.

### Write tests instead of clicking

Classic unit tests (`go test`, `pytest`, `npm test`) call the handler function directly. They check business logic, **not the MCP protocol contract**: handshake, schema validation, error codes, structured results, progress, cancellation. That is exactly where the bugs hide that only show up when a real LLM calls the tool.

With declarative **`.mcp` test scripts** you test the server the way a client sees it: as a black box over the real transport (`stdio`, SSE, Streamable HTTP), with variables, type coercion and assertions, without boilerplate or SDK mocks. The scripts run as a gate in CI and Taskfiles (exit code, `--format json`).

```mcp
call_tool generate_image prompt:"a red flower"
set_var size $.size
assert_equals $size "512x512"

expect_error call_tool generate_image prompt:""
assert_tool_error
```

### Know early what the next spec revision requires

`mcp-tester` is regularly updated to the latest MCP specification; what it checks as of which date is recorded in the [spec coverage](docs/SPEC_COVERAGE.md). Testing your servers with it shows required changes before users or hosts trip over them, and flags **outdated features** such as an old protocol version, removed methods like `ping`, or retired error codes.

We write many MCP servers ourselves, for very different applications. That is where `mcp-tester` saves the most time: one run of the test scripts after a spec or SDK update shows which servers need changes and where the problems are.

Testing early pays off most for new features. Mistakes in **authorization** (OAuth, tokens, headers) are not just annoying, they can be dangerous and expensive. `mcp-tester` runs the full OAuth 2.1 flow and checks the Streamable HTTP transport rules with `http-check`.

### Check third-party and closed-source servers before you enable them

Even for MCP servers whose source you do not know, an analysis makes sense **before** you give them to a "real" LLM. `inspect` shows protocol version, capabilities, tools with schemas and annotations (e.g. whether a tool only reads), icons and anything unusual; `http-check` shows how cleanly the transport is implemented; `skills --verify` shows which instructions (skills) the server hands to the model, which tools they want pre-approved (`allowed-tools`) and whether the content matches the published digests.

Our own internal harness shows exactly these details before an MCP server is enabled. For users it is a very helpful basis for the decision: use it, yes or no?

### Finds bugs in SDKs, too

Because `mcp-tester` checks the server from the outside against the specification, it finds more than bugs in your own code. While adapting to spec 2026-07-28 we also found bugs and gaps in the SDK we use, for example a Base64-encoded `Mcp-Name` header that the official Go SDK v1.8.0 wrongly rejects.

## Key Features

- **True client perspective** over `stdio`, SSE and **Streamable HTTP**, with profiles in `mcp-tester.yml`.
- **Scripting engine** (`.mcp`): tools, tasks, completion, elicitation, sampling, roots and notifications are scriptable, with assertions and an exit code for CI.
- **MCP Apps**: list and check a server's interactive UIs (`apps`): tool linkage, `ui://` resources, external domains (CSP) and requested permissions such as camera or microphone – important before enabling a server.
- **Auth extensions**: client credentials (machine-to-machine) and enterprise login via ID-JAG; `auth-check` shows which flows a server offers.
- **Skills extension**: list and verify a server's skills (`skills --verify`): manifest, digests, frontmatter, naming rules. The Go package [`pkg/mcpskills`](pkg/mcpskills) serves skills from a directory.
- **Tasks extension**: start long-running tool calls as tasks, poll, cancel, supply input. For server authors, the Go package [`pkg/mcptasks`](pkg/mcptasks) adds the extension to servers built on the official go-sdk.
- **Server inspector** (`inspect`): spec check, best practices and quality score; `--min-score` as a CI gate.
- **HTTP conformance** (`http-check`): required headers, error codes, Origin, sessions per spec 2026-07-28.
- **Authorization**: bearer tokens, custom headers and the OAuth 2.1 flow (PKCE, Protected Resource Metadata, `iss` validation).
- **Strict result checks**: every result is checked against the `outputSchema`, as strictly as the official TypeScript SDK.
- **Current specification**: protocol 2026-07-28 with fallback to older revisions; coverage documented with dates.
- **Raw mode** for deep debugging of non-conforming servers.

## FAQ

**`mcp-tester` reports errors – what now?**
Our recommendation is clear: test against the current specification and take the findings seriously. The output names the method, field, error code and the violated rule. Modern LLMs such as Claude or Gemini usually turn that into concrete fix suggestions quickly – just hand them the output (`--format json` works well).

**Do I have to reach a quality score of 100/100?**
No. The score is based on our own experience and assessments; a value below 100 does not automatically call for a change to the MCP server. We still think the number is very useful information, which is why it is there. **Protocol errors** (`inspect`) and **FAIL** (`http-check`) are different: they violate MUST rules of the specification, and real clients fail on them.

---

## The "Everything" Test Server

This project includes a reference server (`cmd/test-server`) that demonstrates the MCP protocol: tools with output schemas, resources and templates, prompts, completion, logging, progress, elicitation, sampling, roots, notifications via `subscriptions/listen` and `x-mcp-header`. With `-addr :8080` it runs over HTTP, with `-auth` additionally OAuth-protected by a built-in test authorization server.

---

## Usage

### 1. Installation

**Via Go (Direct from GitHub):**
```bash
go install github.com/hmsoft0815/mlc_mcptester/cmd/mcp-tester@latest
```
*Note: Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`.*

**Getting Started:**
After installation, you can add your first server and test it immediately:
```bash
# Add a server profile
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"

# List available tools
mcp-tester list -p my-server
```

**Via Curl (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/hmsoft0815/mlc_mcptester/main/scripts/install.sh | bash
```

**Manual Build:**
```bash
git clone https://github.com/hmsoft0815/mlc_mcptester.git
cd mlc_mcptester
task all            # Builds the tester and reference server into the bin/ folder
```

> **Clone without `--recursive`.** The repository points at two
> submodules (`.mlcai`, `mlcprodweb`) that live on an internal server —
> internal documentation and the product page. Neither is needed to
> build; `git clone --recursive` fails without access to that server.

<img src="docs/assets/inspect-and-test.png" alt="mcp-tester inspect and test against an MCP server" width="820">

*Real output: the inspector and a test run against the
[mlc OpticScript](https://mlcgo.eu/products/mlc-opticscript/) MCP server.*

### 2. Commands (Excerpt)

#### Profile Management
Manage different server configurations directly via the CLI:
```bash
mcp-tester profile add my-server -c "npx -y @modelcontextprotocol/server-everything"
mcp-tester profile list
mcp-tester profile disable my-server
mcp-tester profile delete my-server
```

#### Protected servers (authorization)
For HTTP transports. Static credentials via flags or profile, otherwise the spec's OAuth 2.1 flow (PKCE, Protected Resource Metadata, `iss` validation; client registration via Client ID Metadata Document, pre-registered client or dynamic):
```bash
# Bearer token or any header
mcp-tester list -u https://example.com/mcp --bearer "$TOKEN"
mcp-tester list -u https://example.com/mcp -H "X-Api-Key: $KEY"

# OAuth: the URL is printed (or opened with --oauth-browser)
mcp-tester inspect -u https://example.com/mcp --oauth
mcp-tester inspect -u https://example.com/mcp --oauth --oauth-client-id my-client

# Without a browser, for authorization servers without user interaction (CI, test-server -auth)
mcp-tester inspect -u http://127.0.0.1:8080/mcp --oauth --oauth-auto

# Auth extensions: machine-to-machine and enterprise login (ID token → ID-JAG)
mcp-tester list -u https://example.com/mcp --oauth-client-credentials --oauth-client-id svc --oauth-client-secret "$SECRET"
mcp-tester list -u https://example.com/mcp --oauth-enterprise --idp-issuer https://idp.example --idp-client-id app \
  --id-token "$ID_TOKEN" --oauth-client-id app

# Which flows does the server offer, and is the metadata right?
mcp-tester auth-check -u https://example.com/mcp
```
In a profile: `headers:` and `bearer:`, `${VAR}` is expanded from the environment. `./bin/test-server -addr :8080 -auth` starts a protected test server with a built-in authorization server (static token `test-token`).

#### HTTP conformance (`http-check`)
Checks a Streamable HTTP endpoint with hand-built requests against the transport rules of spec 2026-07-28: required headers (`MCP-Protocol-Version`, `Mcp-Method`, `Mcp-Name`, Base64 values, `Mcp-Param-*` from `x-mcp-header`), error codes with HTTP status (`-32020`, `-32022`, 404/`-32601`), Origin validation (403), 405 for GET/DELETE and no sessions. MUST violations: FAIL and exit 1, SHOULD violations: WARN.
```bash
mcp-tester http-check -u https://example.com/mcp --bearer "$TOKEN"
mcp-tester http-check -u http://127.0.0.1:8080/mcp --format json
```

#### Server Inspection
Analyze a server for quality (metadata, prompts, structure):
```bash
# Using a profile from mcp-tester.yml
mcp-tester inspect --profile local

# Direct call without a configuration file
mcp-tester inspect -c "npx -y @modelcontextprotocol/server-everything"

# As a CI gate: exit 1 on protocol errors or a score below the threshold
mcp-tester inspect -p local --min-score 90
```

#### Tools, Resources & Prompts
```bash
mcp-tester list -p local
mcp-tester ping -p local
mcp-tester logging debug -p local
mcp-tester prompts get code_review --args '{"file_path": "main.go"}' -p local
mcp-tester complete prompt:code_review file_path ma -p local
```

#### Test Scripts (Automation)
Execute complex test scenarios:
```bash
./bin/mcp-tester test --script tests/03_variables_and_math.mcp --profile local -v
```

---

## Documentation

- [The MCP Handbook (Online)](https://mlcgo.eu/books/mcp-handbuch/) — A comprehensive introduction and reference to Model Context Protocol (German).
- [Scripting Reference (EN)](docs/SCRIPTING.md) — Detailed documentation of the test grammar.
- [Scripting Referenz (DE)](docs/SCRIPTING.de.md) — Detailed documentation of the test grammar (German).
- [Changelog](CHANGELOG.md) — Changes per version (Added/Changed/Fixed).
- [Spec Coverage (EN)](docs/SPEC_COVERAGE.md) — Which features of the current MCP specification are checked, with date.

---

## License

This project is licensed under the [MIT License](LICENSE).

---
*Copyright Michael Lechner - 2026-03-09*