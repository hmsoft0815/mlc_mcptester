package httpcheck

import "testing"

func TestXMCPHeaders(t *testing.T) {
	obj := func(props map[string]any) map[string]any {
		return map[string]any{"type": "object", "properties": props}
	}
	str := func(header string) map[string]any { return map[string]any{"type": "string", "x-mcp-header": header} }

	valid, problems := XMCPHeaders(obj(map[string]any{
		"region": str("Region"),
		"query":  map[string]any{"type": "string"},
		"nested": obj(map[string]any{"tenant": str("Tenant")}),
	}))
	if len(problems) != 0 || len(valid) != 2 {
		t.Fatalf("valid schema: valid %v, problems %v", valid, problems)
	}

	for name, schema := range map[string]map[string]any{
		"empty":        obj(map[string]any{"a": str("")}),
		"not a token":  obj(map[string]any{"a": str("Re gion")}),
		"duplicate":    obj(map[string]any{"a": str("Region"), "b": str("region")}),
		"number type":  obj(map[string]any{"a": map[string]any{"type": "number", "x-mcp-header": "N"}}),
		"inside items": obj(map[string]any{"list": map[string]any{"type": "array", "items": str("Item")}}),
		"inside oneOf": map[string]any{"type": "object", "oneOf": []any{obj(map[string]any{"a": str("A")})}},
	} {
		if _, problems := XMCPHeaders(schema); len(problems) == 0 {
			t.Errorf("%s: no problem reported", name)
		}
	}
}
