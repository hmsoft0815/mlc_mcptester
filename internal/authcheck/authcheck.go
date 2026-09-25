// Package authcheck discovers how an MCP server over HTTP is protected
// (Protected Resource Metadata, authorization server metadata) and checks it
// against the authorization rules of spec 2026-07-28 and the auth extensions
// (ext-auth: OAuth client credentials, enterprise-managed authorization).
package authcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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

var resourceMetadataParam = regexp.MustCompile(`resource_metadata="([^"]+)"`)

// Discover finds the Protected Resource Metadata (from the 401 challenge,
// else the well-known URIs) and the first authorization server's metadata.
// It returns nil, nil if the endpoint does not require authorization.
func Discover(ctx context.Context, endpoint string, c *http.Client) (*Discovery, error) {
	if c == nil {
		c = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return nil, nil
	}

	d := &Discovery{Challenge: resp.Header.Get("WWW-Authenticate")}
	var candidates []string
	if m := resourceMetadataParam.FindStringSubmatch(d.Challenge); m != nil {
		candidates = append(candidates, m[1])
	}
	candidates = append(candidates, wellKnown(endpoint, "oauth-protected-resource")...)
	for _, u := range candidates {
		if meta, err := getJSON(ctx, c, u); err == nil {
			d.MetadataURL, d.resourceMeta = u, meta
			break
		}
	}
	if d.resourceMeta == nil {
		return d, fmt.Errorf("no Protected Resource Metadata found (tried %s)", strings.Join(candidates, ", "))
	}
	d.Resource, _ = d.resourceMeta["resource"].(string)
	if servers, ok := d.resourceMeta["authorization_servers"].([]any); ok {
		for _, s := range servers {
			if str, ok := s.(string); ok {
				d.AuthServers = append(d.AuthServers, str)
			}
		}
	}
	if issuer := d.AuthServer(); issuer != "" {
		for _, u := range authServerMetadataURLs(issuer) {
			if meta, err := getJSON(ctx, c, u); err == nil {
				d.ASMetadataURL, d.asMeta = u, meta
				break
			}
		}
	}
	return d, nil
}

// wellKnown returns the path-inserted and root well-known URIs (RFC 8615 / 9728).
func wellKnown(resource, name string) []string {
	u, err := url.Parse(resource)
	if err != nil {
		return nil
	}
	root := u.Scheme + "://" + u.Host + "/.well-known/" + name
	if p := strings.TrimSuffix(u.Path, "/"); p != "" {
		return []string{root + p, root}
	}
	return []string{root}
}

// authServerMetadataURLs lists the RFC 8414 and OIDC discovery URLs in the
// order the spec prescribes.
func authServerMetadataURLs(issuer string) []string {
	u, err := url.Parse(issuer)
	if err != nil {
		return nil
	}
	base := u.Scheme + "://" + u.Host
	p := strings.TrimSuffix(u.Path, "/")
	if p == "" {
		return []string{base + "/.well-known/oauth-authorization-server", base + "/.well-known/openid-configuration"}
	}
	return []string{
		base + "/.well-known/oauth-authorization-server" + p,
		base + "/.well-known/openid-configuration" + p,
		base + p + "/.well-known/openid-configuration",
	}
}

func getJSON(ctx context.Context, c *http.Client, u string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
	var m map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", u, err)
	}
	return m, nil
}

// Check runs discovery and all checks against endpoint.
func Check(ctx context.Context, endpoint string, c *http.Client) *Report {
	rep := &Report{Endpoint: endpoint}
	add := func(name string, status Status, format string, args ...any) {
		rep.Results = append(rep.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
	}

	d, err := Discover(ctx, endpoint, c)
	if d == nil && err == nil {
		add("authorization", Info, "the endpoint answers without authorization (no 401)")
		return rep
	}
	rep.Protected = true
	rep.Discovery = d
	if d == nil {
		add("unauthorized request", Fail, "%v", err)
		return rep
	}
	rep.ResourceMeta, rep.AuthServer = d.resourceMeta, d.asMeta

	// 401 challenge (RFC 9728 §5.1)
	switch {
	case !strings.HasPrefix(strings.ToLower(d.Challenge), "bearer"):
		add("WWW-Authenticate", Warn, "401 without a Bearer challenge (%q); clients fall back to the well-known URI", d.Challenge)
	case resourceMetadataParam.MatchString(d.Challenge):
		add("WWW-Authenticate", Pass, "Bearer with resource_metadata")
	default:
		add("WWW-Authenticate", Info, "Bearer without resource_metadata; clients use the well-known URI")
	}
	if err != nil {
		add("Protected Resource Metadata", Fail, "%v", err)
		return rep
	}
	add("Protected Resource Metadata", Pass, "%s", d.MetadataURL)

	// resource and authorization_servers (RFC 9728, MCP authorization)
	if d.Resource == "" {
		add("resource", Fail, "Protected Resource Metadata has no resource")
	} else if canonical(d.Resource) != canonical(endpoint) && !strings.HasPrefix(canonical(endpoint), canonical(d.Resource)) {
		add("resource", Warn, "resource %q does not identify the endpoint %q; tokens are audience-bound to it (RFC 8707)", d.Resource, endpoint)
	} else {
		add("resource", Pass, "%s", d.Resource)
	}
	if len(d.AuthServers) == 0 {
		add("authorization_servers", Fail, "no authorization server advertised")
		return rep
	}
	add("authorization_servers", Pass, "%s", strings.Join(d.AuthServers, ", "))
	if d.asMeta == nil {
		add("authorization server metadata", Fail, "none found for %s (RFC 8414 / OIDC discovery)", d.AuthServer())
		return rep
	}
	add("authorization server metadata", Pass, "%s", d.ASMetadataURL)

	m := d.asMeta
	if iss, _ := m["issuer"].(string); strings.TrimSuffix(iss, "/") != strings.TrimSuffix(d.AuthServer(), "/") {
		add("issuer", Fail, "metadata issuer %q differs from the advertised authorization server %q (RFC 8414 §3.3)", iss, d.AuthServer())
	} else {
		add("issuer", Pass, "%s", iss)
	}
	if s, _ := m["token_endpoint"].(string); s == "" {
		add("token_endpoint", Fail, "missing")
	}

	grants := stringList(m["grant_types_supported"])
	if len(grants) == 0 {
		grants = []string{"authorization_code", "implicit"} // RFC 8414 default
	}
	authMethods := stringList(m["token_endpoint_auth_methods_supported"])
	if len(authMethods) == 0 {
		authMethods = []string{"client_secret_basic"} // RFC 8414 default
	}

	// Authorization code with PKCE: the default flow of the core spec
	authCode := slices.Contains(grants, "authorization_code") && m["authorization_endpoint"] != nil
	if authCode {
		if slices.Contains(stringList(m["code_challenge_methods_supported"]), "S256") {
			add("PKCE (S256)", Pass, "code_challenge_methods_supported includes S256")
		} else {
			add("PKCE (S256)", Fail, "code_challenge_methods_supported lacks S256; MCP clients must refuse to proceed")
		}
		if b, _ := m["authorization_response_iss_parameter_supported"].(bool); b {
			add("iss in authorization response", Pass, "RFC 9207 supported")
		} else {
			add("iss in authorization response", Warn, "authorization_response_iss_parameter_supported is not true; SHOULD per spec 2026-07-28 (RFC 9207)")
		}
	}
	registration := []string{}
	if b, _ := m["client_id_metadata_document_supported"].(bool); b {
		registration = append(registration, "Client ID Metadata Documents")
	}
	if m["registration_endpoint"] != nil {
		registration = append(registration, "dynamic registration (deprecated since 2026-07-28)")
	}
	rep.Flows = append(rep.Flows, Flow{
		Name: "Authorization code + PKCE", Supported: authCode, Flag: "--oauth",
		Detail: "registration: " + orNone(registration) + "; pre-registered clients: --oauth-client-id",
	})

	// ext-auth: OAuth client credentials
	cc := slices.Contains(grants, "client_credentials")
	ccDetail := "token endpoint auth: " + strings.Join(authMethods, ", ")
	if cc {
		if !slices.Contains(authMethods, "private_key_jwt") && !slices.Contains(authMethods, "client_secret_basic") {
			add("client credentials: auth methods", Fail, "token_endpoint_auth_methods_supported must include private_key_jwt or client_secret_basic (ext-auth)")
		} else {
			add("client credentials: auth methods", Pass, "%s", strings.Join(authMethods, ", "))
		}
		if slices.Contains(authMethods, "private_key_jwt") && m["token_endpoint_auth_signing_alg_values_supported"] == nil {
			add("client credentials: signing algorithms", Fail, "private_key_jwt without token_endpoint_auth_signing_alg_values_supported (ext-auth)")
		}
	}
	rep.Flows = append(rep.Flows, Flow{Name: "Client credentials (ext-auth)", Supported: cc, Flag: "--oauth-client-credentials", Detail: ccDetail})

	// ext-auth: enterprise-managed authorization (ID-JAG)
	ema := slices.Contains(stringList(m["authorization_grant_profiles_supported"]), "urn:ietf:params:oauth:grant-profile:id-jag")
	if ema && !slices.Contains(grants, "urn:ietf:params:oauth:grant-type:jwt-bearer") && m["grant_types_supported"] != nil {
		add("enterprise: jwt-bearer grant", Warn, "the id-jag profile is advertised, but grant_types_supported lacks urn:ietf:params:oauth:grant-type:jwt-bearer")
	}
	rep.Flows = append(rep.Flows, Flow{Name: "Enterprise-managed (ID-JAG, ext-auth)", Supported: ema, Flag: "--oauth-enterprise"})
	return rep
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
