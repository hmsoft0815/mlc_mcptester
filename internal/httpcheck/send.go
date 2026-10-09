package httpcheck

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
)

// unsupportedVersion is the future protocol version the checks send.
const unsupportedVersion = "2099-12-31"

// send builds a request that is valid except for what req changes.
func (c *Checker) send(ctx context.Context, req request) (*response, error) {
	httpMethod := req.httpMethod
	if httpMethod == "" {
		httpMethod = http.MethodPost
	}
	var body io.Reader
	headers := map[string]string{}
	if httpMethod == http.MethodPost {
		data, err := postBody(req, headers)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	maps.Copy(headers, req.headers)

	httpReq, err := http.NewRequestWithContext(ctx, httpMethod, c.Endpoint, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		if v != "" {
			httpReq.Header.Set(k, v)
		}
	}
	return c.do(httpReq)
}

// postBody returns the JSON-RPC body of a POST and sets the headers that
// belong to it.
func postBody(req request, headers map[string]string) ([]byte, error) {
	params := maps.Clone(req.params)
	if params == nil {
		params = map[string]any{}
	}
	version := Revision
	if v, ok := req.headers["MCP-Protocol-Version"]; ok && v == unsupportedVersion {
		version = v // unsupported version: header and body agree
	}
	if !req.noMeta {
		params["_meta"] = map[string]any{
			"io.modelcontextprotocol/protocolVersion":    version,
			"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "mcp-tester", "version": "http-check"},
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}
	}
	data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": req.rpcMethod, "params": params})
	if err != nil {
		return nil, err
	}
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json, text/event-stream"
	headers["MCP-Protocol-Version"] = version
	headers["Mcp-Method"] = req.rpcMethod
	if name, ok := req.params["name"].(string); ok {
		headers["Mcp-Name"] = name
	}
	return data, nil
}

// do sends the request and reads at most 1 MiB of the answer.
func (c *Checker) do(httpReq *http.Request) (*response, error) {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	res := &response{status: httpResp.StatusCode, header: httpResp.Header, body: data}
	res.rpc = parseRPC(httpResp.Header.Get("Content-Type"), data)
	return res, nil
}

// parseRPC extracts the JSON-RPC response from a JSON body or the last data
// event of an SSE body.
func parseRPC(contentType string, body []byte) *rpcMessage {
	candidates := [][]byte{body}
	if strings.HasPrefix(contentType, "text/event-stream") {
		candidates = nil
		sc := bufio.NewScanner(bytes.NewReader(body))
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data:"); ok {
				candidates = append(candidates, []byte(strings.TrimSpace(data)))
			}
		}
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		var msg rpcMessage
		if json.Unmarshal(candidates[i], &msg) == nil && (msg.Result != nil || msg.Error != nil) {
			return &msg
		}
	}
	return nil
}

func describe(res *response) string {
	if res.rpc != nil && res.rpc.Error != nil {
		return fmt.Sprintf("HTTP %d with error %d (%s)", res.status, res.rpc.Error.Code, res.rpc.Error.Message)
	}
	if res.rpc != nil {
		return fmt.Sprintf("HTTP %d with a result", res.status)
	}
	text := strings.TrimSpace(string(res.body))
	if len(text) > 80 {
		text = text[:80] + "…"
	}
	return fmt.Sprintf("HTTP %d %q", res.status, text)
}
