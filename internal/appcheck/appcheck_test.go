package appcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const page = "<!DOCTYPE html><html><body>app</body></html>"

type view struct {
	uri, mime, text string
	meta            mcp.Meta
}

// appServer serves tools with the given _meta and UI resources, and connects
// a client that declares the UI capability.
func appServer(t *testing.T, declare bool, tools map[string]mcp.Meta, views ...view) *mcp.ClientSession {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	if declare {
		caps.AddExtension(Extension, map[string]any{})
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "apps", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	for name, meta := range tools {
		mcp.AddTool(s, &mcp.Tool{Name: name, Meta: meta}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
		})
	}
	for _, v := range views {
		v := v
		s.AddResource(&mcp.Resource{URI: v.uri, Name: "view", MIMEType: v.mime}, func(ctx context.Context, r *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: v.uri, MIMEType: v.mime, Text: v.text, Meta: v.meta}}}, nil
		})
	}
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	opts := &mcp.ClientOptions{}
	Declare(opts)
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, opts).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func ui(m map[string]any) mcp.Meta { return mcp.Meta{"ui": m} }

func find(rep *Report, status Status) string {
	var out []string
	for _, r := range rep.Results {
		if r.Status == status {
			out = append(out, r.Name+": "+r.Detail)
		}
	}
	return strings.Join(out, "\n")
}

func TestConformingApp(t *testing.T) {
	cs := appServer(t, true, map[string]mcp.Meta{
		"show":    ui(map[string]any{"resourceUri": "ui://dash"}),
		"refresh": ui(map[string]any{"resourceUri": "ui://dash", "visibility": []string{"app"}}),
	}, view{"ui://dash", MIMEType, page, ui(map[string]any{
		"csp":         map[string]any{"connectDomains": []string{"https://api.example.com"}},
		"permissions": map[string]any{"camera": map[string]any{}},
	})})
	rep := Run(context.Background(), cs)
	if rep.Failed() || find(rep, Warn) != "" {
		t.Fatalf("unexpected findings:\nFAIL:\n%s\nWARN:\n%s", find(rep, Fail), find(rep, Warn))
	}
	if len(rep.Apps) != 1 {
		t.Fatalf("apps: %+v", rep.Apps)
	}
	a := rep.Apps[0]
	if len(a.AppOnly) != 1 || a.AppOnly[0] != "refresh" || len(a.Permissions) != 1 || a.CSP["connectDomains"][0] != "https://api.example.com" {
		t.Errorf("app info: %+v", a)
	}
}

func TestBrokenApps(t *testing.T) {
	cs := appServer(t, false, map[string]mcp.Meta{
		"not_ui":    ui(map[string]any{"resourceUri": "https://example.com/x"}),
		"missing":   ui(map[string]any{"resourceUri": "ui://missing"}),
		"wrongmime": ui(map[string]any{"resourceUri": "ui://plain", "visibility": []string{"user"}}),
		"old":       {"ui/resourceUri": "ui://nodoctype"},
	},
		view{"ui://plain", "text/html", page, nil},
		view{"ui://nodoctype", MIMEType, "<html><body>x</body></html>", ui(map[string]any{"csp": map[string]any{"connectDomains": []string{"*"}}})},
	)
	rep := Run(context.Background(), cs)
	fails, warns := find(rep, Fail), find(rep, Warn)
	for _, want := range []string{"ui:// scheme", "MUST exist", "mimeType", `unknown value "user"`} {
		if !strings.Contains(fails, want) {
			t.Errorf("no FAIL mentioning %q:\n%s", want, fails)
		}
	}
	for _, want := range []string{"deprecated", "DOCTYPE", "not an origin", "does not declare"} {
		if !strings.Contains(warns, want) {
			t.Errorf("no WARN mentioning %q:\n%s", want, warns)
		}
	}
}
