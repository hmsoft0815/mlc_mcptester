package client

import "testing"

func TestParseCompleteRef(t *testing.T) {
	r, err := ParseCompleteRef("prompt:persona_developer")
	if err != nil || r.Type != "ref/prompt" || r.Name != "persona_developer" {
		t.Errorf("prompt ref = %+v, %v", r, err)
	}
	// The URI keeps its own colons
	r, err = ParseCompleteRef("resource:file:///logs/{name}.log")
	if err != nil || r.Type != "ref/resource" || r.URI != "file:///logs/{name}.log" {
		t.Errorf("resource ref = %+v, %v", r, err)
	}
	for _, bad := range []string{"", "prompt", "prompt:", "tool:x"} {
		if _, err := ParseCompleteRef(bad); err == nil {
			t.Errorf("ParseCompleteRef(%q) accepted an invalid reference", bad)
		}
	}
}
