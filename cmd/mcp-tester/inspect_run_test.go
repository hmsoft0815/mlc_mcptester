package main

import (
	"context"
	"strings"
	"testing"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func noop(ctx context.Context, r *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{}, nil, nil
}

var objectSchema = map[string]any{"type": "object"}

// inspectServer runs inspect against an in-memory server built by setup.
func inspectServer(t *testing.T, textOnly map[string]bool, setup func(s *mcp.Server)) *inspector {
	t.Helper()
	return inspectServerWith(t, nil, textOnly, setup)
}

func inspectServerWith(t *testing.T, opts *mcp.ServerOptions, textOnly map[string]bool, setup func(s *mcp.Server)) *inspector {
	t.Helper()
	oldFormat := format
	format = "json" // no text header on stdout
	t.Cleanup(func() { format = oldFormat })

	s := mcp.NewServer(&mcp.Implementation{Name: "inspected", Version: "1"}, opts)
	setup(s)
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	// No command and no URL: the second connection for the tool order check
	// cannot be made, so that check is skipped
	in := newInspector(context.Background(), cs, "", "", textOnly)
	in.run()
	return in
}

func withPrompt(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{Name: "p", Description: "A prompt"}, func(ctx context.Context, r *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{}, nil
	})
}

func recommendationsMention(in *inspector, text string) bool {
	return strings.Contains(strings.Join(in.recommendations, "\n"), text)
}

func TestInspectCleanServer(t *testing.T) {
	in := inspectServer(t, nil, func(s *mcp.Server) {
		withPrompt(s)
		mcp.AddTool(s, &mcp.Tool{Name: "read", Title: "Read", Description: "Reads", InputSchema: objectSchema,
			OutputSchema: objectSchema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, noop)
	})
	if in.report.Score != 100 || len(in.recommendations) != 0 || len(in.protocolErrors) != 0 {
		t.Errorf("score %d, recommendations %v, errors %v", in.report.Score, in.recommendations, in.protocolErrors)
	}
	if in.report.ToolsFound != 1 || in.report.PromptsFound != 1 || in.declaredOutputSchemas != 1 {
		t.Errorf("report %+v", in.report)
	}
}

func TestInspectQualityDeductions(t *testing.T) {
	// No prompts (-20), no title (-1), no description (-5), no output schema (-1)
	in := inspectServer(t, nil, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{Name: "bare", InputSchema: objectSchema}, noop)
	})
	if in.report.Score != 73 {
		t.Errorf("score %d, want 73; recommendations %v", in.report.Score, in.recommendations)
	}
	for _, want := range []string{"bare"} {
		if !recommendationsMention(in, want) {
			t.Errorf("no recommendation mentions %q: %v", want, in.recommendations)
		}
	}
}

func TestInspectTextOnlyAndHintsPerTool(t *testing.T) {
	setup := func(s *mcp.Server) {
		withPrompt(s)
		for _, name := range []string{"markdown", "source"} {
			mcp.AddTool(s, &mcp.Tool{Name: name, Title: name, Description: "Text", InputSchema: objectSchema}, noop)
		}
	}
	summary := inspectServer(t, nil, setup)
	if summary.report.Score != 98 || len(summary.recommendations) != 1 {
		t.Errorf("summary: score %d, recommendations %v (want one line, 98)", summary.report.Score, summary.recommendations)
	}

	exempt := inspectServer(t, map[string]bool{"markdown": true}, setup)
	if exempt.report.Score != 99 || len(exempt.report.TextOnlyTools) != 1 || exempt.report.TextOnlyTools[0] != "markdown" {
		t.Errorf("--text-only: score %d, textOnlyTools %v", exempt.report.Score, exempt.report.TextOnlyTools)
	}
	if !strings.Contains(strings.Join(exempt.infos, "\n"), "markdown") {
		t.Errorf("no INFO names the exempt tool: %v", exempt.infos)
	}

	hintsPerTool = true
	defer func() { hintsPerTool = false }()
	perTool := inspectServer(t, nil, setup)
	if len(perTool.recommendations) != 2 {
		t.Errorf("--hints-per-tool: %v, want one hint per tool", perTool.recommendations)
	}
}

func TestInspectSafetyBonusDoesNotHideSpecViolations(t *testing.T) {
	// Ten read-only tools earn the full bonus (+20); a declared Tasks
	// extension without tasks/* methods is a MUST violation (-10) that the
	// bonus must not offset
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	in := inspectServerWith(t, &mcp.ServerOptions{Capabilities: caps}, nil, func(s *mcp.Server) {
		withPrompt(s)
		for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
			mcp.AddTool(s, &mcp.Tool{Name: name, Title: name, Description: "d", InputSchema: objectSchema, OutputSchema: objectSchema,
				Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, noop)
		}
	})
	if in.report.Score != 90 || !recommendationsMention(in, "Tasks extension") {
		t.Errorf("score %d, want 90 (100 + bonus capped at 100, then -10); recommendations %v", in.report.Score, in.recommendations)
	}
}

func TestVerdict(t *testing.T) {
	cmd := &cobra.Command{}
	in := &inspector{report: InspectionReport{Score: 80}}
	minScore = 90
	defer func() { minScore = 0 }()
	if err := in.verdict(cmd); err == nil || !cmd.SilenceUsage {
		t.Errorf("score below --min-score: %v", err)
	}
	minScore = 0
	in.protocolErrors = []string{"tools/list failed"}
	if err := in.verdict(&cobra.Command{}); err == nil {
		t.Error("protocol errors must fail")
	}
	in.protocolErrors = nil
	if err := in.verdict(&cobra.Command{}); err != nil {
		t.Errorf("clean report: %v", err)
	}
}

func TestTextOnlySet(t *testing.T) {
	oldFlag, oldProfile := textOnlyTools, profile
	defer func() { textOnlyTools, profile = oldFlag, oldProfile }()
	textOnlyTools, profile = []string{"a"}, "p"
	got := textOnlySet(&Config{Profiles: map[string]Profile{"p": {TextOnly: []string{"b"}}}})
	if !got["a"] || !got["b"] || len(got) != 2 {
		t.Errorf("textOnlySet: %v", got)
	}
	if got := textOnlySet(nil); !got["a"] || len(got) != 1 {
		t.Errorf("without config: %v", got)
	}
}

func TestSameToolOrder(t *testing.T) {
	a := []*mcp.Tool{{Name: "x"}, {Name: "y"}}
	if !sameToolOrder(a, []*mcp.Tool{{Name: "x"}, {Name: "y"}}) || sameToolOrder(a, []*mcp.Tool{{Name: "y"}, {Name: "x"}}) {
		t.Error("sameToolOrder")
	}
}
