package httpcheck

import "testing"

func TestParseRPC(t *testing.T) {
	msg := parseRPC("application/json", []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32020,"message":"mismatch"}}`))
	if msg == nil || msg.Error == nil || msg.Error.Code != -32020 {
		t.Fatalf("JSON body: got %+v", msg)
	}

	// SSE: notifications first, the response last
	sse := "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{}}\n\n" +
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"supportedVersions\":[\"2026-07-28\"]}}\n\n"
	msg = parseRPC("text/event-stream", []byte(sse))
	if msg == nil || msg.Result == nil {
		t.Fatalf("SSE body: got %+v", msg)
	}

	if parseRPC("text/plain", []byte("Bad Request")) != nil {
		t.Fatal("plain text parsed as JSON-RPC")
	}
}

func TestReportFailed(t *testing.T) {
	r := &Report{Results: []Result{{Status: Pass}, {Status: Warn}, {Status: Skip}}}
	if r.Failed() {
		t.Fatal("warnings must not fail the report")
	}
	r.Results = append(r.Results, Result{Status: Fail})
	if !r.Failed() {
		t.Fatal("a failed check must fail the report")
	}
}
