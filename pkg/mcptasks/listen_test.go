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
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newServer(t *testing.T) *mcp.Server {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	mcp.AddTool(s, &mcp.Tool{Name: "echo"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct {
		Text string `json:"text"`
	}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: a.Text}}}, nil, nil
	})
	if err := mcptasks.Enable(s, mcptasks.NewStore(), "echo"); err != nil {
		t.Fatal(err)
	}
	return s
}

// checkListen: task notifications without the extension are refused with
// -32021, with it the request reaches the SDK (which acknowledges), and other
// requests pass unchanged.
func checkListen(t *testing.T, cs *mcp.ClientSession) {
	t.Helper()
	listen := func(declare bool) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		caps := map[string]any{}
		if declare {
			caps["extensions"] = map[string]any{mcptasks.Extension: map[string]any{}}
		}
		params := map[string]any{"notifications": map[string]any{"taskIds": []string{"x"}}, "_meta": client.RequestMeta(cs, caps)}
		_, err := client.CallRaw(ctx, cs, "subscriptions/listen", params)
		return err
	}
	var rpc *client.RPCError
	if err := listen(false); !errors.As(err, &rpc) || rpc.Code != -32021 {
		t.Errorf("listen for tasks without the extension: %v, want -32021", err)
	}
	if err := listen(true); err != nil {
		t.Errorf("listen with the extension: %v", err)
	}
	res, err := call(t, cs, "tools/call", map[string]any{"name": "echo", "arguments": map[string]any{"text": "hi"}}, false)
	if err != nil || res["content"] == nil {
		t.Errorf("a plain call through the guard: %v %v", res, err)
	}
}

func TestGuardListen(t *testing.T) {
	ct, st := mcp.NewInMemoryTransports()
	if _, err := newServer(t).Connect(context.Background(), mcptasks.GuardListen(st), nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	checkListen(t, cs)
}

func TestGuardListenHandler(t *testing.T) {
	s := newServer(t)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(mcptasks.GuardListenHandler(h))
	defer srv.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: srv.URL, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if !client.IsStateless(cs) {
		t.Fatalf("negotiated %s", cs.InitializeResult().ProtocolVersion)
	}
	checkListen(t, cs)
}
