package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/authcheck"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/auth/extauth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
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
	// ext-auth
	oauthClientCredentials bool
	oauthEnterprise        bool
	idpIssuer              string
	idpClientID            string
	idpClientSecret        string
	idToken                string
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
	f.BoolVar(&oauthClientCredentials, "oauth-client-credentials", false, "Machine-to-machine: OAuth client credentials grant with --oauth-client-id/--oauth-client-secret (ext-auth)")
	f.BoolVar(&oauthEnterprise, "oauth-enterprise", false, "Enterprise-managed authorization: exchange an SSO ID token at the IdP for an ID-JAG (ext-auth)")
	f.StringVar(&idpIssuer, "idp-issuer", "", "Issuer URL of the enterprise IdP (--oauth-enterprise)")
	f.StringVar(&idpClientID, "idp-client-id", "", "Client ID of mcp-tester at the IdP (--oauth-enterprise)")
	f.StringVar(&idpClientSecret, "idp-client-secret", os.Getenv("MCP_TESTER_IDP_SECRET"), "Client secret at the IdP (default $MCP_TESTER_IDP_SECRET)")
	f.StringVar(&idToken, "id-token", os.Getenv("MCP_TESTER_ID_TOKEN"), "ID token from the SSO login at the IdP (default $MCP_TESTER_ID_TOKEN)")
}

// oauthHandlerFor returns the OAuth handler the flags select, or nil.
func oauthHandlerFor(ctx context.Context, endpoint string, httpClient *http.Client) (auth.OAuthHandler, error) {
	selected := 0
	for _, on := range []bool{oauthEnabled, oauthClientCredentials, oauthEnterprise} {
		if on {
			selected++
		}
	}
	switch {
	case selected == 0:
		return nil, nil
	case selected > 1:
		return nil, fmt.Errorf("choose one of --oauth, --oauth-client-credentials, --oauth-enterprise")
	case oauthEnabled:
		return newOAuthHandler()
	}

	mcpCreds, err := preregisteredClient()
	if err != nil {
		return nil, err
	}
	if oauthClientCredentials {
		if mcpCreds.ClientSecretAuth == nil {
			return nil, fmt.Errorf("--oauth-client-credentials needs --oauth-client-secret (the grant requires a confidential client)")
		}
		return extauth.NewClientCredentialsHandler(&extauth.ClientCredentialsHandlerConfig{Credentials: mcpCreds, HTTPClient: httpClient})
	}

	// Enterprise: the MCP authorization server and resource come from the
	// server's Protected Resource Metadata
	if idpIssuer == "" || idpClientID == "" || idToken == "" {
		return nil, fmt.Errorf("--oauth-enterprise needs --idp-issuer, --idp-client-id and --id-token")
	}
	d, err := authcheck.Discover(ctx, endpoint, httpClient)
	if err != nil {
		return nil, fmt.Errorf("discovering the authorization server: %w", err)
	}
	if d == nil || d.AuthServer() == "" {
		return nil, fmt.Errorf("the server at %s advertises no authorization server", endpoint)
	}
	idpCreds := &oauthex.ClientCredentials{ClientID: idpClientID}
	if idpClientSecret != "" {
		idpCreds.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: idpClientSecret}
	}
	token := idToken
	return extauth.NewEnterpriseHandler(&extauth.EnterpriseHandlerConfig{
		IdPIssuerURL:     idpIssuer,
		IdPCredentials:   idpCreds,
		MCPAuthServerURL: d.AuthServer(),
		MCPResourceURI:   d.Resource,
		MCPCredentials:   mcpCreds,
		HTTPClient:       httpClient,
		IDTokenFetcher: func(context.Context) (*oauth2.Token, error) {
			return (&oauth2.Token{}).WithExtra(map[string]any{"id_token": token}), nil
		},
	})
}

// preregisteredClient is the client registered at the MCP authorization server.
func preregisteredClient() (*oauthex.ClientCredentials, error) {
	if oauthClientID == "" {
		return nil, fmt.Errorf("this flow needs a pre-registered client: --oauth-client-id")
	}
	creds := &oauthex.ClientCredentials{ClientID: oauthClientID}
	if oauthClientSecret != "" {
		creds.ClientSecretAuth = &oauthex.ClientSecretAuth{ClientSecret: oauthClientSecret}
	}
	return creds, nil
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

// taskRoutingTransport sets Mcp-Name to params.taskId on tasks/* requests,
// as the Tasks extension requires on Streamable HTTP; the go-sdk does not
// know these methods and sets no Mcp-Name for them.
type taskRoutingTransport struct {
	base http.RoundTripper
}

func (t *taskRoutingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.Body != nil && strings.HasPrefix(req.Header.Get("Mcp-Method"), "tasks/") && req.Header.Get("Mcp-Name") == "" {
		body, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		var msg struct {
			Params struct {
				TaskID string `json:"taskId"`
			} `json:"params"`
		}
		req = req.Clone(req.Context())
		if json.Unmarshal(body, &msg) == nil && msg.Params.TaskID != "" {
			req.Header.Set("Mcp-Name", msg.Params.TaskID)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}
	return t.base.RoundTrip(req)
}

// withTaskRouting wraps client (nil means the default client).
func withTaskRouting(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	wrapped := *client
	wrapped.Transport = &taskRoutingTransport{base: base}
	return &wrapped
}
