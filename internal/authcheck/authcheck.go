// Package authcheck discovers how an MCP server over HTTP is protected
// (Protected Resource Metadata, authorization server metadata) and checks it
// against the authorization rules of spec 2026-07-28 and the auth extensions
// (ext-auth: OAuth client credentials, enterprise-managed authorization).
package authcheck

import (
	"fmt"
	"strings"
)

// Status of one check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL" // a MUST is violated
	Warn Status = "WARN" // a SHOULD is violated or a deprecated mechanism is in use
	Info Status = "INFO"
	Skip Status = "SKIP"
)

// Result is the outcome of one check.
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Flow is an authorization flow the server offers.
type Flow struct {
	Name      string `json:"name"`
	Supported bool   `json:"supported"`
	Detail    string `json:"detail,omitempty"`
	Flag      string `json:"flag,omitempty"` // how mcp-tester uses it
}

// Report is the outcome of a check run.
type Report struct {
	Endpoint     string         `json:"endpoint"`
	Protected    bool           `json:"protected"`
	Discovery    *Discovery     `json:"discovery,omitempty"`
	Flows        []Flow         `json:"flows,omitempty"`
	Results      []Result       `json:"results"`
	AuthServer   map[string]any `json:"authServerMetadata,omitempty"`
	ResourceMeta map[string]any `json:"resourceMetadata,omitempty"`
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

// Discovery is what a client learns before authorizing.
type Discovery struct {
	Challenge     string   `json:"challenge,omitempty"` // WWW-Authenticate of the 401
	MetadataURL   string   `json:"resourceMetadataUrl"`
	Resource      string   `json:"resource"`
	AuthServers   []string `json:"authorizationServers"`
	ASMetadataURL string   `json:"authServerMetadataUrl,omitempty"`

	resourceMeta map[string]any
	asMeta       map[string]any
}

// AuthServer returns the first advertised authorization server.
func (d *Discovery) AuthServer() string {
	if len(d.AuthServers) == 0 {
		return ""
	}
	return d.AuthServers[0]
}

func canonical(u string) string {
	return strings.TrimSuffix(strings.ToLower(u), "/")
}

func stringList(v any) []string {
	var out []string
	if list, ok := v.([]any); ok {
		for _, x := range list {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func orNone(list []string) string {
	if len(list) == 0 {
		return "none advertised"
	}
	return strings.Join(list, ", ")
}
