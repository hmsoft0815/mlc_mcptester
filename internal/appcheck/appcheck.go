// Package appcheck checks the server side of MCP Apps (io.modelcontextprotocol/ui,
// SEP-1865, stable 2026-01-26): tools that link a ui:// resource, the resources
// themselves (MIME type, HTML) and their security metadata (CSP domains,
// permissions), which is what a host shows before it renders an app.
package appcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Extension is the identifier of the MCP Apps extension.
const Extension = "io.modelcontextprotocol/ui"

// MIMEType is the only content type the extension defines so far.
const MIMEType = "text/html;profile=mcp-app"

// Status of one check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL" // a MUST is violated
	Warn Status = "WARN" // a SHOULD is violated, deprecated or overly broad
	Info Status = "INFO"
)

// Result is the outcome of one check.
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// AppInfo summarizes one UI resource and what it asks the host for.
type AppInfo struct {
	URI         string              `json:"uri"`
	Tools       []string            `json:"tools"`         // tools rendering with it
	AppOnly     []string            `json:"appOnlyTools"`  // hidden from the model
	CSP         map[string][]string `json:"csp,omitempty"` // connectDomains, resourceDomains, …
	Permissions []string            `json:"permissions"`   // camera, microphone, …
	Domain      string              `json:"domain,omitempty"`
	Bytes       int                 `json:"bytes"`
}

// Report is the outcome of a check run.
type Report struct {
	Declared bool      `json:"declared"` // server lists the extension in its capabilities
	Apps     []AppInfo `json:"apps"`
	Results  []Result  `json:"results"`
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == Fail {
			return true
		}
	}
	return false
}

// Declare adds the host capability to client options, so servers that
// register UI tools only for capable hosts expose them.
func Declare(opts *mcp.ClientOptions) {
	if opts.Capabilities == nil {
		opts.Capabilities = &mcp.ClientCapabilities{RootsV2: &mcp.RootCapabilities{ListChanged: true}}
	}
	opts.Capabilities.AddExtension(Extension, map[string]any{"mimeTypes": []string{MIMEType}})
}

// uiMeta is _meta.ui of a tool.
type uiMeta struct {
	ResourceURI string   `json:"resourceUri"`
	Visibility  []string `json:"visibility"`
	hasVis      bool
}

func toolUI(t *mcp.Tool) (uiMeta, bool, bool) {
	var m uiMeta
	deprecated := false
	if raw, ok := t.Meta["ui"]; ok {
		data, _ := json.Marshal(raw)
		_ = json.Unmarshal(data, &m)
		var probe map[string]any
		_ = json.Unmarshal(data, &probe)
		_, m.hasVis = probe["visibility"]
	}
	if m.ResourceURI == "" {
		if s, ok := t.Meta["ui/resourceUri"].(string); ok {
			m.ResourceURI, deprecated = s, true
		}
	}
	return m, m.ResourceURI != "", deprecated
}

// Run checks every tool that links a UI resource.
func Run(ctx context.Context, session *mcp.ClientSession) *Report {
	rep := &Report{}
	add := func(name string, status Status, format string, args ...any) {
		rep.Results = append(rep.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
	}
	_, rep.Declared = session.InitializeResult().Capabilities.Extensions[Extension]

	tools, err := client.ListAllTools(ctx, session)
	if err != nil {
		add("tools/list", Fail, "%v", err)
		return rep
	}
	listed := map[string]*mcp.Resource{}
	if resources, err := client.ListAllResources(ctx, session); err == nil {
		for _, r := range resources {
			listed[r.URI] = r
		}
	}

	apps := map[string]*AppInfo{}
	for _, t := range tools {
		m, ok, deprecated := toolUI(t)
		if !ok {
			continue
		}
		label := "tool " + t.Name
		if deprecated {
			add(label+" _meta", Warn, `_meta["ui/resourceUri"] is deprecated and removed before GA; use _meta.ui.resourceUri`)
		}
		if !strings.HasPrefix(m.ResourceURI, "ui://") {
			add(label+" resourceUri", Fail, "%q must use the ui:// scheme", m.ResourceURI)
			continue
		}
		for _, v := range m.Visibility {
			if v != "model" && v != "app" {
				add(label+" visibility", Fail, "unknown value %q (model, app)", v)
			}
		}
		if m.hasVis && len(m.Visibility) == 0 {
			add(label+" visibility", Warn, "empty visibility: the tool is neither visible to the model nor callable by the app")
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
	if len(apps) == 0 {
		if rep.Declared {
			add("UI tools", Warn, "the server declares %s but no tool links a ui:// resource", Extension)
		} else {
			add("UI tools", Info, "no tool links a UI resource (the server does not use MCP Apps)")
		}
		return rep
	}

	if !rep.Declared {
		add("extension declared", Warn, "tools link UI resources, but the server does not declare %s in its capabilities", Extension)
	}

	uris := make([]string, 0, len(apps))
	for u := range apps {
		uris = append(uris, u)
	}
	sort.Strings(uris)
	for _, u := range uris {
		app := apps[u]
		checkResource(ctx, session, app, listed[u], add)
		rep.Apps = append(rep.Apps, *app)
	}

	for u, r := range listed {
		if strings.HasPrefix(u, "ui://") && apps[u] == nil {
			add("resource "+u, Info, "listed UI resource that no tool links")
		}
		if strings.HasPrefix(u, "ui://") && r.MIMEType != "" && r.MIMEType != MIMEType {
			add("resource "+u+" listing", Warn, "mimeType %q in resources/list, SHOULD be %s", r.MIMEType, MIMEType)
		}
	}
	return rep
}

var doctype = regexp.MustCompile(`(?i)^\s*(<!--.*?-->\s*)*<!doctype\s+html\s*>`)

func checkResource(ctx context.Context, session *mcp.ClientSession, app *AppInfo, listing *mcp.Resource, add func(string, Status, string, ...any)) {
	label := "resource " + app.URI
	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: app.URI})
	if err != nil {
		add(label, Fail, "resources/read failed: %v (the linked resource MUST exist)", err)
		return
	}
	if len(res.Contents) == 0 {
		add(label, Fail, "resources/read returned no contents")
		return
	}
	c := res.Contents[0]
	var problems []string
	if c.MIMEType != MIMEType {
		problems = append(problems, fmt.Sprintf("mimeType %q, MUST be %s", c.MIMEType, MIMEType))
	}
	html := c.Text
	if html == "" && c.Blob != nil {
		html = string(c.Blob)
	}
	app.Bytes = len(html)
	switch {
	case html == "":
		problems = append(problems, "no content in text or blob")
	case !strings.Contains(strings.ToLower(html), "<html"):
		problems = append(problems, "content is not an HTML document (no <html> element)")
	}
	if len(problems) > 0 {
		add(label, Fail, "%s", strings.Join(problems, "; "))
	} else {
		add(label, Pass, "%s, %d bytes", MIMEType, len(html))
	}
	if html != "" && !doctype.MatchString(html) {
		add(label+" doctype", Warn, "no <!DOCTYPE html>; the content MUST be a valid HTML5 document")
	}
	if c.URI != app.URI {
		add(label+" uri", Warn, "content uri %q differs from the requested %q", c.URI, app.URI)
	}

	// _meta.ui on the content item wins over the listing entry
	var meta map[string]any
	if listing != nil {
		meta, _ = listing.Meta["ui"].(map[string]any)
	}
	if m, ok := c.Meta["ui"].(map[string]any); ok {
		meta = m
	}
	checkUIMeta(label, meta, app, add)
}

var origin = regexp.MustCompile(`^(https?|wss?)://(\*\.)?[A-Za-z0-9.-]+(:\d+)?$`)

var knownPermissions = []string{"camera", "microphone", "geolocation", "clipboardWrite"}

// checkUIMeta records and checks the security metadata a host enforces.
func checkUIMeta(label string, meta map[string]any, app *AppInfo, add func(string, Status, string, ...any)) {
	if meta == nil {
		add(label+" CSP", Info, "no _meta.ui: the host applies its restrictive default (no external connections or resources)")
		return
	}
	if csp, ok := meta["csp"].(map[string]any); ok {
		for _, key := range []string{"connectDomains", "resourceDomains", "frameDomains", "baseUriDomains"} {
			list, ok := csp[key].([]any)
			if !ok {
				continue
			}
			for _, v := range list {
				d, _ := v.(string)
				app.CSP[key] = append(app.CSP[key], d)
				switch {
				case !origin.MatchString(d):
					add(label+" csp."+key, Warn, "%q is not an origin (scheme://host[:port], optionally *.host)", d)
				case strings.HasPrefix(d, "http://") && !strings.Contains(d, "localhost") && !strings.Contains(d, "127.0.0.1"):
					add(label+" csp."+key, Warn, "%q is unencrypted http", d)
				}
			}
		}
	}
	if perms, ok := meta["permissions"].(map[string]any); ok {
		for p := range perms {
			app.Permissions = append(app.Permissions, p)
			if !slices.Contains(knownPermissions, p) {
				add(label+" permissions", Warn, "unknown permission %q (%s)", p, strings.Join(knownPermissions, ", "))
			}
		}
		sort.Strings(app.Permissions)
	}
	if d, ok := meta["domain"].(string); ok {
		app.Domain = d
	}
	if b, ok := meta["prefersBorder"]; ok {
		if _, isBool := b.(bool); !isBool {
			add(label+" prefersBorder", Warn, "prefersBorder must be a boolean")
		}
	}
}
