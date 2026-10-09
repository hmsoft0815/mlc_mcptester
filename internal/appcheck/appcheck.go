// Package appcheck checks the server side of MCP Apps (io.modelcontextprotocol/ui,
// SEP-1865, stable 2026-01-26): tools that link a ui:// resource, the resources
// themselves (MIME type, HTML) and their security metadata (CSP domains,
// permissions), which is what a host shows before it renders an app.
package appcheck

import (
	"encoding/json"
	"fmt"

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

func (r *Report) add(name string, status Status, format string, args ...any) {
	r.Results = append(r.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
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
		//lint:ignore SA1019 roots are deprecated since 2026-07-28 (SEP-2577) but still used by servers and regular before; remove with T-20260927-05
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
