package authcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// protectedServer serves a 401 MCP endpoint, Protected Resource Metadata and
// authorization server metadata; asMeta is modified by the test.
func protectedServer(t *testing.T, asMeta func(base string) map[string]any) string {
	t.Helper()
	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+base+`/.well-known/oauth-protected-resource"`)
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"resource": base + "/mcp", "authorization_servers": []string{base}})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(asMeta(base))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL
	return base + "/mcp"
}

func goodMeta(base string) map[string]any {
	return map[string]any{
		"issuer": base, "authorization_endpoint": base + "/authorize", "token_endpoint": base + "/token",
		"grant_types_supported":                          []string{"authorization_code", "client_credentials"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"client_secret_basic"},
		"authorization_response_iss_parameter_supported": true,
		"authorization_grant_profiles_supported":         []string{"urn:ietf:params:oauth:grant-profile:id-jag"},
	}
}

func fails(rep *Report) string {
	var out []string
	for _, r := range rep.Results {
		if r.Status == Fail {
			out = append(out, r.Name+": "+r.Detail)
		}
	}
	return strings.Join(out, "\n")
}

func TestConformingSetup(t *testing.T) {
	rep := Check(context.Background(), protectedServer(t, goodMeta), nil)
	if rep.Failed() {
		t.Fatalf("unexpected failures:\n%s", fails(rep))
	}
	for _, f := range rep.Flows {
		if !f.Supported {
			t.Errorf("flow %q not detected", f.Name)
		}
	}
}

func TestViolations(t *testing.T) {
	for name, tt := range map[string]struct {
		change func(m map[string]any)
		want   string
	}{
		"no S256":           {func(m map[string]any) { m["code_challenge_methods_supported"] = []string{"plain"} }, "PKCE"},
		"issuer mismatch":   {func(m map[string]any) { m["issuer"] = "https://other.example" }, "issuer"},
		"cc auth methods":   {func(m map[string]any) { m["token_endpoint_auth_methods_supported"] = []string{"client_secret_post"} }, "client credentials"},
		"no token_endpoint": {func(m map[string]any) { delete(m, "token_endpoint") }, "token_endpoint"},
	} {
		t.Run(name, func(t *testing.T) {
			endpoint := protectedServer(t, func(base string) map[string]any {
				m := goodMeta(base)
				tt.change(m)
				return m
			})
			rep := Check(context.Background(), endpoint, nil)
			if !strings.Contains(fails(rep), tt.want) {
				t.Errorf("want a failure about %q, got:\n%s", tt.want, fails(rep))
			}
		})
	}
}

func TestUnprotected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	rep := Check(context.Background(), srv.URL, nil)
	if rep.Protected || rep.Failed() {
		t.Errorf("unprotected endpoint reported as protected or failed: %+v", rep.Results)
	}
}
