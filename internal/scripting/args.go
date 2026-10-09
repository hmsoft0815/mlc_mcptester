package scripting

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func findTool(tools []*mcp.Tool, name string) *mcp.Tool {
	for _, t := range tools {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// inputProperties returns the top-level properties of the tool's input
// schema and their names, sorted; both are empty for an unknown tool.
func inputProperties(tool *mcp.Tool) (map[string]any, []string) {
	properties := map[string]any{}
	if tool != nil {
		if schema, ok := tool.InputSchema.(map[string]any); ok {
			if props, ok := schema["properties"].(map[string]any); ok {
				properties = props
			}
		}
	}
	return properties, slices.Sorted(maps.Keys(properties))
}

// buildToolArgs turns script arguments into tool arguments: name:value pairs
// go to their field, the rest fill the remaining fields in name order. For a
// known tool, an unknown name or a leftover value is an error.
func buildToolArgs(tool *mcp.Tool, name string, args []string) (map[string]any, error) {
	properties, propNames := inputProperties(tool)
	toolArgs := map[string]any{}
	var positional []string
	for _, arg := range args {
		if key, val, ok := namedArg(arg); ok {
			if propSchema, ok := properties[key].(map[string]any); ok {
				toolArgs[key] = convertValue(val, propSchema)
				continue
			}
			// A typo in a name must not land silently in another field
			if tool != nil {
				return nil, &scriptError{fmt.Errorf("unknown argument %q: %s takes %s (call_tool_raw sends arguments unchecked)",
					key, name, strings.Join(propNames, ", "))}
			}
		}
		positional = append(positional, arg)
	}
	used := fillPositional(toolArgs, properties, propNames, positional)
	if tool != nil && used < len(positional) {
		return nil, &scriptError{fmt.Errorf("too many arguments for %s: %q has no field left (fields: %s)",
			name, positional[used], strings.Join(propNames, ", "))}
	}
	return toolArgs, nil
}

// fillPositional assigns positional values to the fields not set by name, in
// name order, and returns how many it used.
func fillPositional(toolArgs, properties map[string]any, propNames, positional []string) int {
	used := 0
	for _, propName := range propNames {
		if _, set := toolArgs[propName]; set || used >= len(positional) {
			continue
		}
		propSchema, _ := properties[propName].(map[string]any)
		toolArgs[propName] = convertValue(positional[used], propSchema)
		used++
	}
	return used
}
