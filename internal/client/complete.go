package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ParseCompleteRef turns "prompt:<name>" or "resource:<uri template>" into a
// completion reference.
func ParseCompleteRef(ref string) (*mcp.CompleteReference, error) {
	kind, target, ok := strings.Cut(ref, ":")
	if !ok || target == "" {
		return nil, fmt.Errorf("invalid reference %q: use prompt:<name> or resource:<uri>", ref)
	}
	switch kind {
	case "prompt":
		return &mcp.CompleteReference{Type: "ref/prompt", Name: target}, nil
	case "resource":
		return &mcp.CompleteReference{Type: "ref/resource", URI: target}, nil
	}
	return nil, fmt.Errorf("invalid reference kind %q: use prompt:<name> or resource:<uri>", kind)
}

// Complete asks the server for completion candidates of one argument.
// contextArgs are previously resolved arguments; nil if there are none.
func Complete(ctx context.Context, session *mcp.ClientSession, ref, arg, value string, contextArgs map[string]string) (*mcp.CompleteResult, error) {
	r, err := ParseCompleteRef(ref)
	if err != nil {
		return nil, err
	}
	params := &mcp.CompleteParams{
		Ref:      r,
		Argument: mcp.CompleteParamsArgument{Name: arg, Value: value},
	}
	if len(contextArgs) > 0 {
		params.Context = &mcp.CompleteContext{Arguments: contextArgs}
	}
	return session.Complete(ctx, params)
}
