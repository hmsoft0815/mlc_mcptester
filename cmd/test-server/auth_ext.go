package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Test fixtures for the auth extensions (ext-auth): OAuth client credentials
// and enterprise-managed authorization (ID-JAG). Not for production.
const (
	// TestClientID / TestClientSecret are pre-registered for client credentials
	// and for the JWT bearer grant.
	TestClientID     = "test-client"
	TestClientSecret = "test-secret"
	// TestIDToken is the only ID token the built-in test IdP accepts in a
	// token exchange ("the user logged in via SSO").
	TestIDToken = "test-id-token"

	grantTypeJWTBearer     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	grantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	tokenTypeIDJAG         = "urn:ietf:params:oauth:token-type:id-jag"
	tokenTypeIDToken       = "urn:ietf:params:oauth:token-type:id_token"
	idJAGProfile           = "urn:ietf:params:oauth:grant-profile:id-jag"
)

// idjagKey signs the ID-JAGs of the test IdP (HS256; a real IdP uses its JWKS).
var idjagKey = []byte(randomString(32))

// clientAuth returns the client id if the request authenticates the
// pre-registered test client (client_secret_basic or client_secret_post).
func clientAuth(r *http.Request) (string, bool) {
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	return id, id == TestClientID && secret == TestClientSecret
}

// clientCredentialsGrant: machine-to-machine token for a pre-registered client.
func (a *authServer) clientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	if _, ok := clientAuth(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	if res := r.PostForm.Get("resource"); res != "" && res != a.resource {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_target", "error_description": "resource must be " + a.resource})
		return
	}
	a.issueToken(w)
}

// jwtBearerGrant accepts an ID-JAG from the test IdP (enterprise-managed authorization).
func (a *authServer) jwtBearerGrant(w http.ResponseWriter, r *http.Request) {
	if _, ok := clientAuth(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	claims, err := verifyJWT(r.PostForm.Get("assertion"))
	switch {
	case err != nil:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": err.Error()})
		return
	case claims["aud"] != a.issuer:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "aud must be " + a.issuer})
		return
	case claims["resource"] != nil && claims["resource"] != a.resource:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "resource must be " + a.resource})
		return
	}
	a.issueToken(w) // audience-restricted to this MCP server
}

// registerIdP mounts a minimal enterprise IdP at /idp: metadata and a token
// endpoint that exchanges TestIDToken for an ID-JAG.
func (a *authServer) registerIdP(mux *http.ServeMux) {
	idp := a.issuer + "/idp"
	meta := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer":                                idp,
			"token_endpoint":                        idp + "/token",
			"authorization_endpoint":                idp + "/authorize", // the SSO login (not simulated here)
			"response_types_supported":              []string{"code"},
			"code_challenge_methods_supported":      []string{"S256"},
			"grant_types_supported":                 []string{"authorization_code", grantTypeTokenExchange},
			"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		})
	}
	mux.HandleFunc("/idp/.well-known/openid-configuration", meta)
	mux.HandleFunc("/.well-known/oauth-authorization-server/idp", meta)
	mux.HandleFunc("/idp/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.PostForm.Get("grant_type") != grantTypeTokenExchange {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
			return
		}
		clientID, ok := clientAuth(r)
		f := r.PostForm
		switch {
		case !ok:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		case f.Get("requested_token_type") != tokenTypeIDJAG || f.Get("subject_token_type") != tokenTypeIDToken:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "only id_token -> id-jag"})
			return
		case f.Get("subject_token") != TestIDToken:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "unknown subject token"})
			return
		case f.Get("audience") != a.issuer:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_target", "error_description": "audience must be " + a.issuer})
			return
		}
		now := time.Now()
		jag := signJWT(map[string]any{"typ": "oauth-id-jag+jwt", "alg": "HS256"}, map[string]any{
			"jti": randomString(8), "iss": idp, "sub": "test-user", "aud": f.Get("audience"),
			"resource": f.Get("resource"), "client_id": clientID, "scope": f.Get("scope"),
			"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"issued_token_type": tokenTypeIDJAG, "access_token": jag, "token_type": "N_A", "expires_in": 300,
		})
	})
}

func signJWT(header, claims map[string]any) string {
	enc := func(v any) string {
		data, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(data)
	}
	unsigned := enc(header) + "." + enc(claims)
	mac := hmac.New(sha256.New, idjagKey)
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verifyJWT(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("assertion is not a JWT")
	}
	mac := hmac.New(sha256.New, idjagKey)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("invalid assertion signature")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	if exp, _ := claims["exp"].(float64); int64(exp) < time.Now().Unix() {
		return nil, fmt.Errorf("assertion expired")
	}
	return claims, nil
}
