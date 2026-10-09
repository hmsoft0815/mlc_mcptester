package scripting

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

// schemaAllowsType checks if the given schema allows targetType.
// It handles string types, type slices (e.g. ["null", "array"]), and anyOf/oneOf.
func schemaAllowsType(schema map[string]any, targetType string) bool {
	if schema == nil {
		return false
	}
	if slices.Contains(typeNames(schema["type"]), targetType) {
		return true
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		for _, sub := range subSchemas(schema[key]) {
			if schemaAllowsType(sub, targetType) {
				return true
			}
		}
	}
	return false
}

// typeNames returns the names in a schema's "type": one string or a list.
func typeNames(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []string:
		return t
	case []any:
		var names []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		return names
	}
	return nil
}

// subSchemas returns the object entries of an anyOf/oneOf list.
func subSchemas(v any) []map[string]any {
	list, _ := v.([]any)
	var subs []map[string]any
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			subs = append(subs, m)
		}
	}
	return subs
}

// convertValue converts a string value to the type specified in the schema.
func convertValue(val string, schema map[string]any) any {
	trimmed := strings.TrimSpace(val)
	if v, ok := convertJSON(trimmed, schema); ok {
		return v
	}
	if v, ok := convertScalar(trimmed, schema); ok {
		return v
	}
	return val
}

// convertJSON parses a JSON array or object when the schema allows that
// type, or when the value has its syntax and the schema does not take a string.
func convertJSON(trimmed string, schema map[string]any) (any, bool) {
	takesString := schemaAllowsType(schema, "string")
	if enclosed(trimmed, "[", "]") && (schemaAllowsType(schema, "array") || !takesString) {
		var result []any
		if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
			return result, true
		}
	}
	if enclosed(trimmed, "{", "}") && (schemaAllowsType(schema, "object") || !takesString) {
		var result map[string]any
		if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
			return result, true
		}
	}
	return nil, false
}

func enclosed(s, open, close string) bool {
	return strings.HasPrefix(s, open) && strings.HasSuffix(s, close)
}

// convertScalar converts to the first of integer, number and boolean that
// the schema allows and the value parses as.
func convertScalar(trimmed string, schema map[string]any) (any, bool) {
	if schemaAllowsType(schema, "integer") {
		if i, err := strconv.Atoi(trimmed); err == nil {
			return i, true
		}
	}
	if schemaAllowsType(schema, "number") {
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return f, true
		}
	}
	if schemaAllowsType(schema, "boolean") {
		switch strings.ToLower(trimmed) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return nil, false
}
