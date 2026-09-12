package scripting

import (
	"encoding/json"
	"strconv"
	"strings"
)

// schemaAllowsType checks if the given schema allows targetType.
// It handles string types, type slices (e.g. ["null", "array"]), and anyOf/oneOf.
func schemaAllowsType(schema map[string]any, targetType string) bool {
	if schema == nil {
		return false
	}
	switch t := schema["type"].(type) {
	case string:
		if t == targetType {
			return true
		}
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && s == targetType {
				return true
			}
		}
	case []string:
		for _, s := range t {
			if s == targetType {
				return true
			}
		}
	}

	for _, key := range []string{"anyOf", "oneOf"} {
		if subSchemas, ok := schema[key].([]any); ok {
			for _, sub := range subSchemas {
				if subMap, ok := sub.(map[string]any); ok {
					if schemaAllowsType(subMap, targetType) {
						return true
					}
				}
			}
		}
	}

	return false
}

// convertValue converts a string value to the type specified in the schema.
func convertValue(val string, schema map[string]any) any {
	trimmed := strings.TrimSpace(val)

	// Array conversion: when schema allows array OR when value is JSON array syntax
	if schemaAllowsType(schema, "array") || (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && !schemaAllowsType(schema, "string")) {
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			var result []any
			if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
				return result
			}
		}
	}

	// Object conversion: when schema allows object OR when value is JSON object syntax
	if schemaAllowsType(schema, "object") || (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") && !schemaAllowsType(schema, "string")) {
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			var result map[string]any
			if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
				return result
			}
		}
	}

	if schemaAllowsType(schema, "integer") {
		if i, err := strconv.Atoi(trimmed); err == nil {
			return i
		}
	}

	if schemaAllowsType(schema, "number") {
		if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
			return f
		}
	}

	if schemaAllowsType(schema, "boolean") {
		lower := strings.ToLower(trimmed)
		if lower == "true" || lower == "1" {
			return true
		}
		if lower == "false" || lower == "0" {
			return false
		}
	}

	return val
}
