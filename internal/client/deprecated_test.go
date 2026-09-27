package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDeprecationWatchInspect(t *testing.T) {
	w := &DeprecationWatch{}
	for _, params := range []string{
		`{"name":"x"}`,
		`{"name":"x","inputResponses":{"confirm":{"action":"accept","content":{"ok":true}}}}`,
		`not json`,
	} {
		w.inspect(json.RawMessage(params))
	}
	if used := w.Used(); len(used) != 0 {
		t.Fatalf("elicitation and plain calls reported as deprecated: %v", used)
	}

	w.inspect(json.RawMessage(`{"name":"x","inputResponses":{"llm":{"role":"assistant","model":"m","content":{"type":"text","text":"hi"}}}}`))
	w.inspect(json.RawMessage(`{"taskId":"t","inputResponses":{"r":{"roots":[{"uri":"file:///a"}]}}}`))
	if used := w.Used(); !reflect.DeepEqual(used, []string{"roots", "sampling"}) {
		t.Errorf("Used = %v, want [roots sampling]", used)
	}
}
