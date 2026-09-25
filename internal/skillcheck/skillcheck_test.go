package skillcheck

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var skillsFS = fstest.MapFS{
	"git-workflow/SKILL.md":                 {Data: []byte("---\nname: git-workflow\ndescription: Follow the team's Git conventions. Use when committing.\n---\n# Git\nSee references/branches.md\n")},
	"git-workflow/references/branches.md":   {Data: []byte("main, feature/*, fix/*\n")},
	"acme/billing/refunds/SKILL.md":         {Data: []byte("---\nname: refunds\ndescription: Process refunds per policy.\nlicense: Apache-2.0\nmetadata:\n  version: \"1.0\"\n---\nSteps...\n")},
	"acme/billing/refunds/logo.bin":         {Data: []byte{0xff, 0x00, 0xfe}},
	"acme/billing/refunds/partial/SKILL.md": {Data: []byte("---\nname: partial\ndescription: Partial refunds.\n---\nNested.\n")},
}

func connect(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestConformingServer(t *testing.T) {
	caps := &mcp.ServerCapabilities{}
	mcpskills.Declare(caps, true)
	s := mcp.NewServer(&mcp.Implementation{Name: "skills", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	served, err := mcpskills.Serve(s, skillsFS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(served) != 3 {
		t.Fatalf("served %d skills, want 3", len(served))
	}

	rep := (&Checker{Session: connect(t, s), Verify: true}).Run(context.Background())
	for _, r := range rep.Results {
		if r.Status == Fail {
			t.Errorf("FAIL %s: %s", r.Name, r.Detail)
		}
	}
	if len(rep.Skills) != 3 || !rep.DirectoryRead {
		t.Errorf("skills %d, directoryRead %v", len(rep.Skills), rep.DirectoryRead)
	}
	// The enclosing skill lists the nested skill's files too
	for _, sk := range rep.Skills {
		if sk.Name == "refunds" && sk.Files != 3 {
			t.Errorf("refunds lists %d files, want 3 (nested SKILL.md included)", sk.Files)
		}
	}
}

func TestTamperedServer(t *testing.T) {
	caps := &mcp.ServerCapabilities{}
	mcpskills.Declare(caps, false)
	s := mcp.NewServer(&mcp.Implementation{Name: "bad", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	body := []byte("---\nname: tool-guide\ndescription: Guide.\n---\nBody\n")
	s.AddResource(&mcp.Resource{URI: "skill://guide/SKILL.md", Name: "guide"}, func(ctx context.Context, r *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: r.Params.URI, Text: string(body)}}}, nil
	})
	// Name differs from the path, wrong digest, SKILL.md missing from the manifest, no cache hints
	bad := mcpskills.Skill{
		URI:         "skill://guide/SKILL.md",
		Frontmatter: map[string]any{"name": "tool-guide", "description": "Guide."},
		Resources:   []mcpskills.SkillResource{{URI: "skill://guide/other.md", Digest: "sha256:abc", Size: 3}},
	}
	type listResult struct {
		mcp.ResultBase
		ResultType string            `json:"resultType"`
		Skills     []mcpskills.Skill `json:"skills"`
	}
	type getResult struct {
		mcp.ResultBase
		ResultType string          `json:"resultType"`
		Skill      mcpskills.Skill `json:"skill"`
	}
	if err := mcp.AddReceivingCustomMethod(s, "skills/list", func(ctx context.Context, _ *mcp.ServerSession, p *mcpskills.ListParams) (*listResult, error) {
		return &listResult{ResultType: "complete", Skills: []mcpskills.Skill{bad}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := mcp.AddReceivingCustomMethod(s, "skills/get", func(ctx context.Context, _ *mcp.ServerSession, p *mcpskills.URIParams) (*getResult, error) {
		// Answers every URI, also unknown ones (must be -32602)
		return &getResult{ResultType: "complete", Skill: bad}, nil
	}); err != nil {
		t.Fatal(err)
	}

	rep := (&Checker{Session: connect(t, s), Verify: true}).Run(context.Background())
	var fails []string
	for _, r := range rep.Results {
		if r.Status == Fail {
			fails = append(fails, r.Name+": "+r.Detail)
		}
	}
	all := strings.Join(fails, "\n")
	for _, want := range []string{"must equal the frontmatter name", "digest", "does not list SKILL.md", "cache hints", "unknown skill"} {
		if !strings.Contains(all, want) {
			t.Errorf("no failure mentioning %q; failures:\n%s", want, all)
		}
	}
}

func TestFormat(t *testing.T) {
	for _, name := range []string{"pdf-processing", "a", "x1-y2"} {
		if msg := mcpskills.ValidateName(name); msg != "" {
			t.Errorf("%q rejected: %s", name, msg)
		}
	}
	for _, name := range []string{"", "PDF", "-pdf", "pdf-", "pdf--x", "pdf_x", strings.Repeat("a", 65)} {
		if mcpskills.ValidateName(name) == "" {
			t.Errorf("%q accepted", name)
		}
	}
	fm, err := mcpskills.ParseFrontmatter([]byte("---\nname: x\ndescription: y\n---\nbody"))
	if err != nil || fm["name"] != "x" {
		t.Errorf("ParseFrontmatter: %v, %v", fm, err)
	}
	if _, err := mcpskills.ParseFrontmatter([]byte("no frontmatter")); err == nil {
		t.Error("accepted a SKILL.md without frontmatter")
	}
	if !mcpskills.ValidDigest(mcpskills.Digest([]byte("x"))) || mcpskills.ValidDigest("sha256:ABC") {
		t.Error("digest format check is wrong")
	}
}
