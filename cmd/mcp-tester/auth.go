package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

var (
	headerFlags        []string
	bearerToken        string
	oauthEnabled       bool
	oauthClientID      string
	oauthClientSecret  string
	oauthMetadataURL   string
	oauthCallbackPort  int
	oauthOpenBrowser   bool
	oauthAutoAuthorize bool
)

func init() {
	f := rootCmd.PersistentFlags()
	f.StringArrayVarP(&headerFlags, "header", "H", nil, `HTTP header for HTTP transports, "Name: value" (repeatable)`)
	f.StringVar(&bearerToken, "bearer", os.Getenv("MCP_TESTER_BEARER"), "Bearer token for HTTP transports (default $MCP_TESTER_BEARER)")
	f.BoolVar(&oauthEnabled, "oauth", false, "Authorize via OAuth 2.1 authorization code flow when the server answers 401")
	f.StringVar(&oauthClientID, "oauth-client-id", "", "Pre-registered OAuth client ID (otherwise dynamic client registration)")
	f.StringVar(&oauthClientSecret, "oauth-client-secret", os.Getenv("MCP_TESTER_OAUTH_SECRET"), "Secret of the pre-registered client (default $MCP_TESTER_OAUTH_SECRET)")
	f.StringVar(&oauthMetadataURL, "oauth-client-metadata-url", "", "HTTPS URL of a Client ID Metadata Document (preferred registration method)")
	f.IntVar(&oauthCallbackPort, "oauth-callback-port", 3142, "Local port receiving the authorization redirect")
	f.BoolVar(&oauthOpenBrowser, "oauth-browser", false, "Open the authorization URL in the browser instead of only printing it")
	f.BoolVar(&oauthAutoAuthorize, "oauth-auto", false, "Follow the authorization URL without a browser (for servers that approve without user interaction, e.g. in CI)")
}

// httpClientWithAuth returns an HTTP client adding --header and --bearer to
// every request, or nil if neither is set.
func httpClientWithAuth() (*http.Client, error) {
	headers := http.Header{}
	for _, h := range headerFlags {
		name, value, ok := strings.Cut(h, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf(`invalid --header %q, expected "Name: value"`, h)
		}
		headers.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	if bearerToken != "" {
		headers.Set("Authorization", "Bearer "+bearerToken)
	}
	if len(headers) == 0 {
		return nil, nil
	}
	return &http.Client{Transport: &headerTransport{headers: headers, base: http.DefaultTransport}}, nil
}

type headerTransport struct {
	headers http.Header
	base    http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for name, values := range t.headers {
		req.Header[name] = values
	}
	return t.base.RoundTrip(req)
}

// newOAuthHandler builds the SDK's authorization code handler (PKCE, resource
// indicators, RFC 9207 iss validation). Registration is tried in the order the
// spec prefers: Client ID Metadata Document, pre-registration, dynamic.
func newOAuthHandler() (*auth.AuthorizationCodeHandler, error) {
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", oauthCallbackPort)
	cfg := &auth.AuthorizationCodeHandlerConfig{
		RedirectURL:              redirectURL,
		AuthorizationCodeFetcher: fetchAuthorizationCode,
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName:   "mcp-tester",
				RedirectURIs: []string{redirectURL},
			},
		},
	}
	if oauthMetadataURL != "" {
		cfg.ClientIDMetadataDocumentConfig = &auth.ClientIDMetadataDocumentConfig{URL: oauthMetadataURL}
	}
	if oauthClientID != "" {
		cfg.PreregisteredClient = &oauthex.ClientCredentials{ClientID: oauthClientID}
		if oauthClientSecret != "" {
			cfg.PreregisteredClient.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: oauthClientSecret}
		}
	}
	return auth.NewAuthorizationCodeHandler(cfg)
}

// fetchAuthorizationCode sends the user to the authorization URL and waits for
// the redirect to the local callback.
func fetchAuthorizationCode(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", oauthCallbackPort))
	if err != nil {
		return nil, fmt.Errorf("listening for the OAuth callback: %w", err)
	}
	results := make(chan *auth.AuthorizationResult, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			http.Error(w, "Authorization failed: "+e, http.StatusBadRequest)
			return
		}
		select {
		case results <- &auth.AuthorizationResult{Code: q.Get("code"), State: q.Get("state"), Iss: q.Get("iss")}:
		default:
		}
		fmt.Fprintln(w, "mcp-tester: authorization received, you can close this window.")
	})}
	go srv.Serve(listener)
	defer srv.Close()

	switch {
	case oauthAutoAuthorize:
		// The authorization server redirects straight to our callback
		resp, err := http.Get(args.URL)
		if err != nil {
			return nil, fmt.Errorf("following the authorization URL: %w", err)
		}
		resp.Body.Close()
	case oauthOpenBrowser:
		fmt.Fprintf(os.Stderr, "Opening browser for authorization: %s\n", args.URL)
		if err := openBrowser(args.URL); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open the browser (%v); open the URL manually.\n", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Open this URL to authorize mcp-tester:\n  %s\n", args.URL)
	}

	select {
	case res := <-results:
		return res, nil
	case <-ctx.Done():
		return nil, errors.Join(errors.New("waiting for the OAuth callback"), ctx.Err())
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
