package mcptasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The go-sdk answers subscriptions/listen itself and drops
// notifications.taskIds, so it cannot refuse a client that asks for task
// notifications without declaring the extension, as the extension requires
// (-32021). GuardListen and GuardListenHandler do that below the SDK.
// Task notifications themselves are not sent (MAY): the acknowledgement
// lists no task ids, and clients poll with tasks/get.

const methodSubscriptionsListen = "subscriptions/listen"

// maxListenBody bounds the part of a request body GuardListenHandler reads;
// a subscriptions/listen request is far smaller.
const maxListenBody = 4 << 20

// missingCapability is the -32021 error naming the extension.
func missingCapability() *jsonrpc.Error {
	return &jsonrpc.Error{
		Code:    codeMissingRequiredCapability,
		Message: "Missing required client capability",
		Data:    json.RawMessage(`{"requiredCapabilities":{"extensions":{"` + Extension + `":{}}}}`),
	}
}

// refuseListen reports a subscriptions/listen request that asks for task
// notifications without declaring the extension.
func refuseListen(msg jsonrpc.Message) (*jsonrpc.Request, bool) {
	req, ok := msg.(*jsonrpc.Request)
	if !ok || req.Method != methodSubscriptionsListen || !req.IsCall() {
		return nil, false
	}
	var p struct {
		Meta          mcp.Meta `json:"_meta"`
		Notifications struct {
			TaskIDs []string `json:"taskIds"`
		} `json:"notifications"`
	}
	if json.Unmarshal(req.Params, &p) != nil {
		return nil, false
	}
	return req, len(p.Notifications.TaskIDs) > 0 && !declaresTasks(p.Meta)
}

// GuardListen wraps a server transport (e.g. stdio) so that such requests
// are refused with -32021 before they reach the SDK.
func GuardListen(t mcp.Transport) mcp.Transport { return guardTransport{t} }

type guardTransport struct{ mcp.Transport }

func (t guardTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return guardConn{conn}, nil
}

type guardConn struct{ mcp.Connection }

func (c guardConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	for {
		msg, err := c.Connection.Read(ctx)
		if err != nil {
			return nil, err
		}
		req, refuse := refuseListen(msg)
		if !refuse {
			return msg, nil
		}
		if err := c.Connection.Write(ctx, &jsonrpc.Response{ID: req.ID, Error: missingCapability()}); err != nil {
			return nil, err
		}
	}
}

// GuardListenHandler wraps a Streamable HTTP handler likewise: such a
// request gets the -32021 error as a JSON response.
func GuardListenHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mcp-Method names the method when present; without it the body tells
		method := r.Header.Get("Mcp-Method")
		if r.Method != http.MethodPost || (method != "" && method != methodSubscriptionsListen) {
			h.ServeHTTP(w, r)
			return
		}
		body, err := peekBody(r)
		if err != nil {
			http.Error(w, "reading the request body failed", http.StatusBadRequest)
			return
		}
		req, refuse := refuseListenBody(body)
		if !refuse {
			h.ServeHTTP(w, r)
			return
		}
		out, err := jsonrpc.EncodeMessage(&jsonrpc.Response{ID: req.ID, Error: missingCapability()})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	})
}

// peekBody reads up to maxListenBody bytes of the request body and puts them
// back in front of the rest, so the next handler gets the whole body.
func peekBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxListenBody))
	if err != nil {
		return nil, err
	}
	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(body), r.Body), r.Body}
	return body, nil
}

// refuseListenBody is refuseListen for a raw body; a body that fills the
// whole peek buffer is no listen request and is not decoded.
func refuseListenBody(body []byte) (*jsonrpc.Request, bool) {
	if len(body) == maxListenBody {
		return nil, false
	}
	msg, err := jsonrpc.DecodeMessage(body)
	if err != nil {
		return nil, false
	}
	return refuseListen(msg)
}
