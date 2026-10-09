package appcheck

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run checks every tool that links a UI resource.
func Run(ctx context.Context, session *mcp.ClientSession) *Report {
	rep := &Report{}
	_, rep.Declared = session.InitializeResult().Capabilities.Extensions[Extension]

	tools, err := client.ListAllTools(ctx, session)
	if err != nil {
		rep.add("tools/list", Fail, "%v", err)
		return rep
	}
	listed := listedResources(ctx, session)
	apps := map[string]*AppInfo{}
	for _, t := range tools {
		rep.linkTool(apps, t)
	}
	if len(apps) == 0 {
		rep.reportNoApps()
		return rep
	}
	if !rep.Declared {
		rep.add("extension declared", Warn, "tools link UI resources, but the server does not declare %s in its capabilities", Extension)
	}
	for _, u := range slices.Sorted(maps.Keys(apps)) {
		app := apps[u]
		checkResource(ctx, session, app, listed[u], rep)
		rep.Apps = append(rep.Apps, *app)
	}
	rep.checkListing(listed, apps)
	return rep
}

// listedResources indexes resources/list by URI; a failing list is no finding.
func listedResources(ctx context.Context, session *mcp.ClientSession) map[string]*mcp.Resource {
	listed := map[string]*mcp.Resource{}
	if resources, err := client.ListAllResources(ctx, session); err == nil {
		for _, r := range resources {
			listed[r.URI] = r
		}
	}
	return listed
}

// linkTool checks the UI metadata of one tool and records it with its app.
func (rep *Report) linkTool(apps map[string]*AppInfo, t *mcp.Tool) {
	m, ok, deprecated := toolUI(t)
	if !ok {
		return
	}
	label := "tool " + t.Name
	if deprecated {
		rep.add(label+" _meta", Warn, `_meta["ui/resourceUri"] is deprecated and removed before GA; use _meta.ui.resourceUri`)
	}
	if !strings.HasPrefix(m.ResourceURI, "ui://") {
		rep.add(label+" resourceUri", Fail, "%q must use the ui:// scheme", m.ResourceURI)
		return
	}
	for _, v := range m.Visibility {
		if v != "model" && v != "app" {
			rep.add(label+" visibility", Fail, "unknown value %q (model, app)", v)
		}
	}
	if m.hasVis && len(m.Visibility) == 0 {
		rep.add(label+" visibility", Warn, "empty visibility: the tool is neither visible to the model nor callable by the app")
	}
	app := apps[m.ResourceURI]
	if app == nil {
		app = &AppInfo{URI: m.ResourceURI, CSP: map[string][]string{}}
		apps[m.ResourceURI] = app
	}
	app.Tools = append(app.Tools, t.Name)
	if m.hasVis && !slices.Contains(m.Visibility, "model") {
		app.AppOnly = append(app.AppOnly, t.Name)
	}
}

func (rep *Report) reportNoApps() {
	if rep.Declared {
		rep.add("UI tools", Warn, "the server declares %s but no tool links a ui:// resource", Extension)
	} else {
		rep.add("UI tools", Info, "no tool links a UI resource (the server does not use MCP Apps)")
	}
}

// checkListing looks at the ui:// entries of resources/list: unlinked ones
// and the MIME type they are listed with.
func (rep *Report) checkListing(listed map[string]*mcp.Resource, apps map[string]*AppInfo) {
	for _, u := range slices.Sorted(maps.Keys(listed)) {
		if !strings.HasPrefix(u, "ui://") {
			continue
		}
		if apps[u] == nil {
			rep.add("resource "+u, Info, "listed UI resource that no tool links")
		}
		if r := listed[u]; r.MIMEType != "" && r.MIMEType != MIMEType {
			rep.add("resource "+u+" listing", Warn, "mimeType %q in resources/list, SHOULD be %s", r.MIMEType, MIMEType)
		}
	}
}
