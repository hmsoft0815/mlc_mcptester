package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientWithAuth(t *testing.T) {
	defer func(h []string, b string) { headerFlags, bearerToken = h, b }(headerFlags, bearerToken)

	headerFlags, bearerToken = nil, ""
	if c, err := httpClientWithAuth(); err != nil || c != nil {
		t.Fatalf("no flags: got client %v, err %v; want nil, nil", c, err)
	}

	headerFlags = []string{"no-colon"}
	if _, err := httpClientWithAuth(); err == nil {
		t.Fatal("accepted a header without colon")
	}

	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Header }))
	defer srv.Close()

	headerFlags, bearerToken = []string{"X-Tenant: acme", "X-Trace:  abc "}, "secret"
	c, err := httpClientWithAuth()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for name, want := range map[string]string{"X-Tenant": "acme", "X-Trace": "abc", "Authorization": "Bearer secret"} {
		if got.Get(name) != want {
			t.Errorf("header %s = %q; want %q", name, got.Get(name), want)
		}
	}
}

func TestTaskRoutingTransport(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.Header.Get("Mcp-Name")+"|"+string(body))
	}))
	defer srv.Close()

	c := withTaskRouting(nil)
	post := func(method, body string) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(body))
		req.Header.Set("Mcp-Method", method)
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	post("tasks/get", `{"params":{"taskId":"t-1"}}`)
	post("tools/list", `{"params":{}}`)

	want := []string{`t-1|{"params":{"taskId":"t-1"}}`, `|{"params":{}}`}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("request %d: got %q, want %q (body must pass unchanged)", i, got[i], want[i])
		}
	}
}
