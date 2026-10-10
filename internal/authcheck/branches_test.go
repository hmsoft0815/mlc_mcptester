package authcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// customServer serves a 401 with the given challenge and the given Protected
// Resource Metadata at the root well-known URI; with asMeta nil the
// authorization server publishes no metadata.
func customServer(t *testing.T, challenge func(base string) string, prm func(base string) map[string]any, asMeta func(base string) map[string]any) string {
	t.Helper()
	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if c := challenge(base); c != "" {
			w.Header().Set("WWW-Authenticate", c)
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	if prm != nil {
		mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(prm(base))
		})
	}
	if asMeta != nil {
		mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(asMeta(base))
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL
	return base + "/mcp"
}

func statusOf(rep *Report, name string) Status {
	for _, r := range rep.Results {
		if r.Name == name {
			return r.Status
		}
	}
	return ""
}

func TestCheckBranches(t *testing.T) {
	prm := func(base string) map[string]any {
		return map[string]any{"resource": base + "/mcp", "authorization_servers": []string{base}}
	}
	bearer := func(string) string { return "Bearer" }
	tests := map[string]struct {
		challenge func(string) string
		prm       func(string) map[string]any
		asMeta    func(string) map[string]any
		want      map[string]Status
	}{
		"no Bearer challenge": {func(string) string { return `Basic realm="x"` }, prm, goodMeta,
			map[string]Status{"WWW-Authenticate": Warn, "Protected Resource Metadata": Pass}},
		"Bearer without resource_metadata": {bearer, prm, goodMeta,
			map[string]Status{"WWW-Authenticate": Info, "resource": Pass}},
		"resource names another endpoint": {bearer, func(base string) map[string]any {
			return map[string]any{"resource": "https://elsewhere.example/api", "authorization_servers": []string{base}}
		}, goodMeta, map[string]Status{"resource": Warn}},
		"no resource": {bearer, func(base string) map[string]any {
			return map[string]any{"authorization_servers": []string{base}}
		}, goodMeta, map[string]Status{"resource": Fail}},
		"no authorization server": {bearer, func(base string) map[string]any {
			return map[string]any{"resource": base + "/mcp"}
		}, goodMeta, map[string]Status{"authorization_servers": Fail, "issuer": ""}},
		"no authorization server metadata": {bearer, prm, nil,
			map[string]Status{"authorization server metadata": Fail, "issuer": ""}},
		"no Protected Resource Metadata": {bearer, nil, nil,
			map[string]Status{"Protected Resource Metadata": Fail, "resource": ""}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rep := Check(context.Background(), customServer(t, tt.challenge, tt.prm, tt.asMeta), nil)
			if !rep.Protected {
				t.Fatal("not seen as protected")
			}
			for check, want := range tt.want {
				if got := statusOf(rep, check); got != want {
					t.Errorf("%s: %q, want %q", check, got, want)
				}
			}
		})
	}
}

func TestCapabilitiesDefaults(t *testing.T) {
	caps := capabilitiesOf(map[string]any{})
	if len(caps.grants) != 2 || caps.grants[0] != "authorization_code" || caps.authMethods[0] != "client_secret_basic" {
		t.Errorf("RFC 8414 defaults: %+v", caps)
	}
}

func TestOpenEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	rep := Check(context.Background(), srv.URL, nil)
	if rep.Protected || statusOf(rep, "authorization") != Info {
		t.Errorf("open endpoint: %+v", rep)
	}
}
