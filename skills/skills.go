// Package skills holds the Agent Skills of this repository, at the
// standard location skills/<name>/SKILL.md, embedded for the binary.
package skills

import _ "embed"

// MCPTester is skills/mcp-tester/SKILL.md, installed by
// "mcp-tester agent-skill install".
//
//go:embed mcp-tester/SKILL.md
var MCPTester string
