package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// StaticToken is always accepted in -auth mode, for tests with --bearer.
const StaticToken = "test-token"

// authServer is a minimal OAuth 2.1 authorization server for testing MCP
// clients: RFC 8414 metadata, dynamic client registration, an authorization
// endpoint that approves every request, and a token endpoint enforcing PKCE
// (S256). It also serves the MCP server's Protected Resource Metadata.
// Nothing here is meant for production.
type authServer struct {
	issuer   string // e.g. http://127.0.0.1:8080
	resource string // the protected MCP endpoint

	mu     sync.Mutex
	codes  map[string]pendingCode // authorization code -> PKCE challenge etc.
	tokens map[string]time.Time   // access token -> expiry
}

type pendingCode struct {
	challenge   string
	redirectURI string
	clientID    string
}

func newAuthServer(issuer, resource string) *authServer {
	return &authServer{
		issuer:   issuer,
		resource: resource,
		codes:    map[string]pendingCode{},
		tokens:   map[string]time.Time{},
	}
}

// register mounts the authorization server and resource metadata endpoints.
func (a *authServer) register(mux *http.ServeMux) {
	prm := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:             a.resource,
		AuthorizationServers: []string{a.issuer},
		ScopesSupported:      []string{"mcp"},
	})
	mux.Handle("/.well-known/oauth-protected-resource", prm)
	mux.Handle("/.well-known/oauth-protected-resource/", prm)
	mux.HandleFunc("/.well-known/oauth-authorization-server", a.metadata)
	mux.HandleFunc("/register", a.registerClient)
	mux.HandleFunc("/authorize", a.authorize)
	mux.HandleFunc("/token", a.token)
}

// protect wraps the MCP handler with bearer token verification.
func (a *authServer) protect(h http.Handler) http.Handler {
	return auth.RequireBearerToken(a.verify, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: a.issuer + "/.well-known/oauth-protected-resource",
	})(h)
}

func (a *authServer) verify(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	if token == StaticToken {
		return &auth.TokenInfo{Scopes: []string{"mcp"}, Expiration: time.Now().Add(time.Hour)}, nil
	}
	a.mu.Lock()
	expiry, ok := a.tokens[token]
	a.mu.Unlock()
	if !ok || time.Now().After(expiry) {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{Scopes: []string{"mcp"}, Expiration: expiry}, nil
}

func (a *authServer) metadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         a.issuer,
		"authorization_endpoint":                         a.issuer + "/authorize",
		"token_endpoint":                                 a.issuer + "/token",
		"registration_endpoint":                          a.issuer + "/register",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"scopes_supported":                               []string{"mcp"},
		"authorization_response_iss_parameter_supported": true,
	})
}

// registerClient implements dynamic client registration (RFC 7591).
func (a *authServer) registerClient(w http.ResponseWriter, r *http.Request) {
	var meta map[string]any
	if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_client_metadata"})
		return
	}
	meta["client_id"] = "client-" + randomString(8)
	meta["client_id_issued_at"] = time.Now().Unix()
	meta["token_endpoint_auth_method"] = "none"
	writeJSON(w, http.StatusCreated, meta)
}

// authorize approves every request and redirects back with a code (RFC 9207 iss included).
func (a *authServer) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	if q.Get("response_type") != "code" || redirectURI == "" || q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		http.Error(w, "invalid authorization request: response_type=code, redirect_uri and an S256 code_challenge are required", http.StatusBadRequest)
		return
	}
	target, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	code := randomString(16)
	a.mu.Lock()
	a.codes[code] = pendingCode{challenge: q.Get("code_challenge"), redirectURI: redirectURI, clientID: q.Get("client_id")}
	a.mu.Unlock()

	params := target.Query()
	params.Set("code", code)
	params.Set("state", q.Get("state"))
	params.Set("iss", a.issuer)
	target.RawQuery = params.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// token exchanges an authorization code for an access token after checking PKCE.
func (a *authServer) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.PostForm.Get("grant_type") != "authorization_code" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
		return
	}
	code := r.PostForm.Get("code")
	a.mu.Lock()
	pending, ok := a.codes[code]
	delete(a.codes, code) // single use
	a.mu.Unlock()

	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	switch {
	case !ok:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "unknown or used code"})
		return
	case base64.RawURLEncoding.EncodeToString(sum[:]) != pending.challenge:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "PKCE verification failed"})
		return
	case r.PostForm.Get("redirect_uri") != pending.redirectURI:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "redirect_uri mismatch"})
		return
	}

	token := randomString(24)
	a.mu.Lock()
	a.tokens[token] = time.Now().Add(time.Hour)
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        "mcp",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)
}
