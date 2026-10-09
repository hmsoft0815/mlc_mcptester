package appcheck

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var doctype = regexp.MustCompile(`(?i)^\s*(<!--.*?-->\s*)*<!doctype\s+html\s*>`)

// checkResource reads a linked UI resource and checks its content and metadata.
func checkResource(ctx context.Context, session *mcp.ClientSession, app *AppInfo, listing *mcp.Resource, rep *Report) {
	label := "resource " + app.URI
	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: app.URI})
	if err != nil {
		rep.add(label, Fail, "resources/read failed: %v (the linked resource MUST exist)", err)
		return
	}
	if len(res.Contents) == 0 {
		rep.add(label, Fail, "resources/read returned no contents")
		return
	}
	c := res.Contents[0]
	checkContent(label, c, app, rep)
	if c.URI != app.URI {
		rep.add(label+" uri", Warn, "content uri %q differs from the requested %q", c.URI, app.URI)
	}

	// _meta.ui on the content item wins over the listing entry
	var meta map[string]any
	if listing != nil {
		meta, _ = listing.Meta["ui"].(map[string]any)
	}
	if m, ok := c.Meta["ui"].(map[string]any); ok {
		meta = m
	}
	checkUIMeta(label, meta, app, rep)
}

// checkContent: the MCP Apps MIME type and an HTML5 document.
func checkContent(label string, c *mcp.ResourceContents, app *AppInfo, rep *Report) {
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
		rep.add(label, Fail, "%s", strings.Join(problems, "; "))
	} else {
		rep.add(label, Pass, "%s, %d bytes", MIMEType, len(html))
	}
	if html != "" && !doctype.MatchString(html) {
		rep.add(label+" doctype", Warn, "no <!DOCTYPE html>; the content MUST be a valid HTML5 document")
	}
}

var origin = regexp.MustCompile(`^(https?|wss?)://(\*\.)?[A-Za-z0-9.-]+(:\d+)?$`)

var knownPermissions = []string{"camera", "microphone", "geolocation", "clipboardWrite"}

// checkUIMeta records and checks the security metadata a host enforces.
func checkUIMeta(label string, meta map[string]any, app *AppInfo, rep *Report) {
	if meta == nil {
		rep.add(label+" CSP", Info, "no _meta.ui: the host applies its restrictive default (no external connections or resources)")
		return
	}
	if csp, ok := meta["csp"].(map[string]any); ok {
		checkCSP(label, csp, app, rep)
	}
	if perms, ok := meta["permissions"].(map[string]any); ok {
		checkPermissions(label, perms, app, rep)
	}
	if d, ok := meta["domain"].(string); ok {
		app.Domain = d
	}
	if b, ok := meta["prefersBorder"]; ok {
		if _, isBool := b.(bool); !isBool {
			rep.add(label+" prefersBorder", Warn, "prefersBorder must be a boolean")
		}
	}
}

// checkCSP records the CSP domains; each must be an origin, and https unless local.
func checkCSP(label string, csp map[string]any, app *AppInfo, rep *Report) {
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
				rep.add(label+" csp."+key, Warn, "%q is not an origin (scheme://host[:port], optionally *.host)", d)
			case strings.HasPrefix(d, "http://") && !strings.Contains(d, "localhost") && !strings.Contains(d, "127.0.0.1"):
				rep.add(label+" csp."+key, Warn, "%q is unencrypted http", d)
			}
		}
	}
}

// checkPermissions records the requested permissions and flags unknown ones.
func checkPermissions(label string, perms map[string]any, app *AppInfo, rep *Report) {
	for _, p := range slices.Sorted(maps.Keys(perms)) {
		app.Permissions = append(app.Permissions, p)
		if !slices.Contains(knownPermissions, p) {
			rep.add(label+" permissions", Warn, "unknown permission %q (%s)", p, strings.Join(knownPermissions, ", "))
		}
	}
}
