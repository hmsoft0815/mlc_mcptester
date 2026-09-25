package main

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// clockHTML is the view of the demo app: it shows the time from the tool result.
const clockHTML = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>Clock</title></head>
<body>
<h1 id="time">…</h1>
<script>
  // Hosts send the tool result as ui/notifications/tool-result (MCP Apps)
  window.addEventListener("message", (e) => {
    const msg = e.data;
    if (msg && msg.method === "ui/notifications/tool-result") {
      const sc = msg.params && msg.params.structuredContent;
      document.getElementById("time").textContent = sc ? sc.time : "";
    }
  });
</script>
</body>
</html>`

// registerApps adds a small MCP App (io.modelcontextprotocol/ui): a tool whose
// result a host renders with the ui://clock view, and an app-only tool.
func registerApps(s *mcp.Server) {
	uiMeta := map[string]any{
		"csp":           map[string]any{"resourceDomains": []string{"https://cdn.jsdelivr.net"}},
		"prefersBorder": true,
	}
	s.AddResource(&mcp.Resource{
		URI:         "ui://clock",
		Name:        "clock",
		Description: "Shows the server time",
		MIMEType:    "text/html;profile=mcp-app",
		Meta:        mcp.Meta{"ui": uiMeta},
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: "ui://clock", MIMEType: "text/html;profile=mcp-app", Text: clockHTML, Meta: mcp.Meta{"ui": uiMeta},
		}}}, nil
	})

	type clock struct {
		Time string `json:"time" jsonschema:"Current server time"`
	}
	now := func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, clock, error) {
		// A meaningful text result also for hosts without MCP Apps
		return nil, clock{Time: time.Now().Format(time.RFC3339)}, nil
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "show_clock",
		Title:       "Show Clock",
		Description: "Returns the server time; UI-capable hosts render it with the ui://clock app",
		Meta:        mcp.Meta{"ui": map[string]any{"resourceUri": "ui://clock"}},
	}, now)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "refresh_clock",
		Title:       "Refresh Clock",
		Description: "Called by the clock app to update itself; hidden from the model",
		Meta:        mcp.Meta{"ui": map[string]any{"resourceUri": "ui://clock", "visibility": []string{"app"}}},
	}, now)
}
