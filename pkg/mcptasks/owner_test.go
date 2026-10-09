package mcptasks_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

// authServer serves the slow tool as a task over HTTP; alice and bob carry
// user ids, the anonymous tokens do not (bound by token digest).
func authServer(t *testing.T) string {
	t.Helper()
	users := map[string]string{"alice-token": "alice", "bob-token": "bob", "anon-1": "", "anon-2": ""}
	verify := func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		user, ok := users[token]
		if !ok {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{UserID: user, Expiration: time.Now().Add(time.Hour)}, nil
	}
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	mcp.AddTool(s, &mcp.Tool{Name: "slow"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	})
	if err := mcptasks.Enable(s, mcptasks.NewStore(), "slow"); err != nil {
		t.Fatal(err)
	}
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(auth.RequireBearerToken(verify, nil)(h))
	t.Cleanup(srv.Close)
	return srv.URL
}

func session(t *testing.T, url, token string) *mcp.ClientSession {
	t.Helper()
	tr := &mcp.StreamableClientTransport{Endpoint: url, HTTPClient: &http.Client{Transport: bearer{token}}, MaxRetries: -1}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), tr, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestTasksBoundToIdentity(t *testing.T) {
	url := authServer(t)
	for _, pair := range [][2]string{{"alice-token", "bob-token"}, {"anon-1", "anon-2"}} {
		t.Run(pair[0], func(t *testing.T) {
			owner, other := session(t, url, pair[0]), session(t, url, pair[1])
			id := start(t, owner, "slow", nil)
			for _, method := range []string{"tasks/get", "tasks/update", "tasks/cancel"} {
				_, err := call(t, other, method, map[string]any{"taskId": id, "inputResponses": map[string]any{}}, true)
				var rpc *client.RPCError
				if !errors.As(err, &rpc) || rpc.Code != -32602 {
					t.Errorf("%s by another identity: %v, want -32602", method, err)
				}
			}
			// The owner still sees the task, and it was not cancelled
			waitFor(t, owner, id, mcptasks.Working)
		})
	}
}
