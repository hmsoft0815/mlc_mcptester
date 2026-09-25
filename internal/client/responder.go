package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Responder answers the input a server asks the client for (elicitation and
// sampling, since 2026-07-28 via multi round-trip requests) from queued,
// prepared answers, and remembers the last request for assertions.
type Responder struct {
	// Strict makes a request without a queued answer fail the call; otherwise
	// elicitations are declined and sampling fails with an explanation.
	Strict bool
	// Out receives one line per answered request.
	Out io.Writer

	mu         sync.Mutex
	elicit     []*mcp.ElicitResult
	samples    []string
	lastElicit *mcp.ElicitParams
	lastSample string
}

// ParseElicitAnswer turns "accept", "decline", "cancel" or "accept:<json>" and
// an optional separate JSON object into an elicitation result.
func ParseElicitAnswer(action, content string) (*mcp.ElicitResult, error) {
	if a, c, ok := strings.Cut(action, ":"); ok && content == "" {
		action, content = a, c
	}
	switch action {
	case "accept", "decline", "cancel":
	default:
		return nil, fmt.Errorf("invalid elicitation action %q: use accept, decline or cancel", action)
	}
	res := &mcp.ElicitResult{Action: action}
	if content != "" {
		if action != "accept" {
			return nil, fmt.Errorf("only accept carries content")
		}
		if err := json.Unmarshal([]byte(content), &res.Content); err != nil {
			return nil, fmt.Errorf("elicitation content must be a JSON object: %w", err)
		}
	}
	return res, nil
}

// QueueElicit adds an answer for the next elicitation request.
func (r *Responder) QueueElicit(res *mcp.ElicitResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.elicit = append(r.elicit, res)
}

// QueueSample adds the model reply for the next sampling request.
func (r *Responder) QueueSample(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples = append(r.samples, text)
}

// LastElicitMessage returns the message of the last elicitation request, or "".
func (r *Responder) LastElicitMessage() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastElicit == nil {
		return ""
	}
	if r.lastElicit.URL != "" {
		return r.lastElicit.Message + " " + r.lastElicit.URL
	}
	return r.lastElicit.Message
}

// LastSamplePrompt returns the text of the last sampling request, or "".
func (r *Responder) LastSamplePrompt() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastSample
}

// Install sets the handlers on opts and declares the matching capabilities:
// roots, elicitation in form and URL mode, and sampling.
func (r *Responder) Install(opts *mcp.ClientOptions) {
	opts.Capabilities = &mcp.ClientCapabilities{
		RootsV2:     &mcp.RootCapabilities{ListChanged: true},
		Elicitation: &mcp.ElicitationCapabilities{Form: &mcp.FormElicitationCapabilities{}, URL: &mcp.URLElicitationCapabilities{}},
	}
	opts.ElicitationHandler = r.handleElicit
	opts.CreateMessageHandler = r.handleSample
}

func (r *Responder) handleElicit(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastElicit = req.Params

	var res *mcp.ElicitResult
	if len(r.elicit) > 0 {
		res, r.elicit = r.elicit[0], r.elicit[1:]
	} else if r.Strict {
		return nil, fmt.Errorf("unexpected elicitation %q: no elicit_response queued", req.Params.Message)
	} else {
		res = &mcp.ElicitResult{Action: "decline"}
	}
	r.logf("[ELICIT] %s → %s%s\n", req.Params.Message, res.Action, contentSuffix(res.Content))
	return res, nil
}

func (r *Responder) handleSample(ctx context.Context, req *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var prompt []string
	for _, m := range req.Params.Messages {
		if t, ok := m.Content.(*mcp.TextContent); ok {
			prompt = append(prompt, t.Text)
		}
	}
	r.lastSample = strings.Join(prompt, "\n")

	if len(r.samples) == 0 {
		return nil, fmt.Errorf("unexpected sampling request %q: no sample_response queued", r.lastSample)
	}
	var text string
	text, r.samples = r.samples[0], r.samples[1:]
	r.logf("[SAMPLE] %s → %s\n", r.lastSample, text)
	return &mcp.CreateMessageResult{
		Role:    "assistant",
		Model:   "mcp-tester",
		Content: &mcp.TextContent{Text: text},
	}, nil
}

func (r *Responder) logf(format string, args ...any) {
	if r.Out != nil {
		fmt.Fprintf(r.Out, format, args...)
	}
}

func contentSuffix(content map[string]any) string {
	if len(content) == 0 {
		return ""
	}
	data, _ := json.Marshal(content)
	return " " + string(data)
}
