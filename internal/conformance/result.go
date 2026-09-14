// Package conformance checks MCP results the way a strict client checks them,
// so a server fails here rather than in someone's editor.
package conformance

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// CheckToolResult checks a tools/call result against the output schema its tool
// declares. result is the result object as it came over the wire.
//
// The specification requires a tool that declares an outputSchema to return
// structuredContent that conforms to it. The official TypeScript SDK — used by
// OpenCode, among others — enforces both: it rejects a result without
// structuredContent with error -32600 and validates one that has it. The Go SDK
// this tester is built on checks neither on the client side, and neither do
// Claude Code or crush, so a server can break every such tool without any of
// them showing it. That happened to codebase-memory: eleven tools, all rejected
// by OpenCode, all passing here.
//
// A result with isError is not checked; the specification exempts it, and so
// does the TypeScript SDK.
func CheckToolResult(outputSchema any, result map[string]any) error {
	if outputSchema == nil {
		return nil
	}
	if isError, _ := result["isError"].(bool); isError {
		return nil
	}

	structured, ok := result["structuredContent"]
	if !ok || structured == nil {
		return fmt.Errorf("the tool declares an outputSchema but the result has no structuredContent; " +
			"strict clients such as OpenCode (official TypeScript SDK) reject this call with -32600")
	}

	resolved, err := resolve(outputSchema)
	if err != nil {
		return fmt.Errorf("the declared outputSchema cannot be used to validate: %w", err)
	}

	// Round-tripped so the validator sees plain JSON values whatever decoded
	// the result: json.RawMessage, Go integers and structs all become what a
	// client parsing the wire format would have.
	instance, err := asJSONValue(structured)
	if err != nil {
		return fmt.Errorf("structuredContent is not valid JSON: %w", err)
	}
	if err := resolved.Validate(instance); err != nil {
		return fmt.Errorf("structuredContent does not match the declared outputSchema: %w", err)
	}
	return nil
}

func resolve(outputSchema any) (*jsonschema.Resolved, error) {
	data, err := json.Marshal(outputSchema)
	if err != nil {
		return nil, err
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return schema.Resolve(nil)
}

func asJSONValue(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	err = json.Unmarshal(data, &out)
	return out, err
}
