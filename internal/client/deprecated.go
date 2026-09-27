package client

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeprecationWatch notes the deprecated client features a server asks for:
// sampling and roots, deprecated since 2026-07-28 (SEP-2577).
//
// It watches the wire, not the SDK: the SDK answers such requests inside its
// multi round-trip loop, behind any middleware. Answers travel in the
// inputResponses of the retried request, and multi round-trip only exists
// from 2026-07-28 on, so servers on older revisions — where both features
// are regular server-to-client requests — are not reported.
type DeprecationWatch struct {
	mu   sync.Mutex
	used map[string]bool
}

// Wrap returns t with every outgoing request inspected.
func (w *DeprecationWatch) Wrap(t mcp.Transport) mcp.Transport {
	return watchTransport{Transport: t, w: w}
}

// Used lists the deprecated features seen so far, sorted.
func (w *DeprecationWatch) Used() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var used []string
	for f := range w.used {
		used = append(used, f)
	}
	sort.Strings(used)
	return used
}

// inspect records the features answered in a request's inputResponses:
// a roots/list result carries "roots", a sampling result "role" and "model".
func (w *DeprecationWatch) inspect(params json.RawMessage) {
	var p struct {
		InputResponses map[string]map[string]json.RawMessage `json:"inputResponses"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	for _, resp := range p.InputResponses {
		_, roots := resp["roots"]
		_, role := resp["role"]
		_, model := resp["model"]
		switch {
		case roots:
			w.note("roots")
		case role && model:
			w.note("sampling")
		}
	}
}

func (w *DeprecationWatch) note(feature string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.used == nil {
		w.used = map[string]bool{}
	}
	w.used[feature] = true
}

type watchTransport struct {
	mcp.Transport
	w *DeprecationWatch
}

func (t watchTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return watchConn{Connection: conn, w: t.w}, nil
}

type watchConn struct {
	mcp.Connection
	w *DeprecationWatch
}

func (c watchConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	if req, ok := msg.(*jsonrpc.Request); ok && len(req.Params) > 0 {
		c.w.inspect(req.Params)
	}
	return c.Connection.Write(ctx, msg)
}
