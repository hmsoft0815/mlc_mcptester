package taskcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer string

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+string(b))
	return http.DefaultTransport.RoundTrip(req)
}

// authServer serves the slow tool as a task over HTTP to two users.
func authServer(t *testing.T, owner mcptasks.OwnerFunc) string {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	mcp.AddTool(s, &mcp.Tool{Name: "slow"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(time.Second):
			return text("done"), nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})
	if err := mcptasks.Enable(s, &mcptasks.Store{PollInterval: 20 * time.Millisecond, Owner: owner}, "slow"); err != nil {
		t.Fatal(err)
	}
	verify := func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		return &auth.TokenInfo{UserID: token, Expiration: time.Now().Add(time.Hour)}, nil
	}
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(mcptasks.GuardListenHandler(auth.RequireBearerToken(verify, nil)(h)))
	t.Cleanup(srv.Close)
	return srv.URL
}

func httpSession(t *testing.T, url, token string) *mcp.ClientSession {
	t.Helper()
	tr := &mcp.StreamableClientTransport{Endpoint: url, HTTPClient: &http.Client{Transport: bearer(token)}, MaxRetries: -1}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), tr, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestBinding(t *testing.T) {
	unbound := func(*mcp.RequestExtra) string { return "" }
	for name, tt := range map[string]struct {
		owner mcptasks.OwnerFunc
		want  Status
	}{"bound": {nil, Pass}, "unbound": {unbound, Fail}} {
		t.Run(name, func(t *testing.T) {
			url := authServer(t, tt.owner)
			rep := (&Checker{
				Session: httpSession(t, url, "alice"), Other: httpSession(t, url, "bob"),
				Tool: "slow", PlainWait: 200 * time.Millisecond, Timeout: 10 * time.Second,
			}).Run(context.Background())
			for _, name := range []string{"tasks/get by another identity", "tasks/cancel by another identity"} {
				if r := result(rep, name); r == nil || (tt.want == Pass && r.Status != Pass) || (tt.want == Fail && r.Status == Pass) {
					t.Errorf("%s: %+v, want %s", name, r, tt.want)
				}
			}
			if tt.want == Fail {
				if r := result(rep, "auth binding"); r == nil || r.Status != Fail {
					t.Errorf("a cancel by another identity went unnoticed: %+v", r)
				}
				return
			}
			noFailures(t, rep)
		})
	}
}
