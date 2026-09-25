package scripting

import (
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Commands that prepare the client's side of multi round-trip requests:
// answers to elicitation and sampling, and the roots offered to the server.

func (r *Runner) responder(i int) (*client.Responder, error) {
	if r.Responder == nil {
		return nil, fmt.Errorf("line %d: this runner cannot answer input requests", i+1)
	}
	return r.Responder, nil
}

// handleElicitResponseCommand runs "elicit_response accept|decline|cancel [json]".
func (r *Runner) handleElicitResponseCommand(i int, parts []string) error {
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("line %d: usage: elicit_response accept|decline|cancel ['{\"key\":\"value\"}']", i+1)
	}
	resp, err := r.responder(i)
	if err != nil {
		return err
	}
	content := ""
	if len(parts) == 3 {
		content = parts[2]
	}
	res, err := client.ParseElicitAnswer(parts[1], content)
	if err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
	resp.QueueElicit(res)
	return nil
}

// handleSampleResponseCommand runs "sample_response <text>".
func (r *Runner) handleSampleResponseCommand(i int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: usage: sample_response <text>", i+1)
	}
	resp, err := r.responder(i)
	if err != nil {
		return err
	}
	resp.QueueSample(strings.Join(parts[1:], " "))
	return nil
}

// handleAddRootCommand runs "add_root <uri> [name]".
func (r *Runner) handleAddRootCommand(i int, parts []string) error {
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("line %d: usage: add_root <uri> [name]", i+1)
	}
	if r.Client == nil {
		return fmt.Errorf("line %d: this runner cannot offer roots", i+1)
	}
	root := &mcp.Root{URI: parts[1]}
	if len(parts) == 3 {
		root.Name = parts[2]
	}
	r.Client.AddRoots(root)
	r.roots = append(r.roots, root)
	return nil
}

// handleAssertElicitedCommand runs "assert_elicited <substring>" against the
// message of the last elicitation request.
func (r *Runner) handleAssertElicitedCommand(i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: assert_elicited <substring>", i+1)
	}
	resp, err := r.responder(i)
	if err != nil {
		return err
	}
	msg := resp.LastElicitMessage()
	if msg == "" {
		return fmt.Errorf("line %d: assertion failed: no elicitation request received", i+1)
	}
	if !strings.Contains(msg, parts[1]) {
		return fmt.Errorf("line %d: assertion failed: elicitation %q does not contain %q", i+1, msg, parts[1])
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, fmt.Sprintf("elicitation contains %q", parts[1])))
	return nil
}

// handleAssertSampledCommand runs "assert_sampled <substring>" against the
// prompt of the last sampling request.
func (r *Runner) handleAssertSampledCommand(i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: assert_sampled <substring>", i+1)
	}
	resp, err := r.responder(i)
	if err != nil {
		return err
	}
	prompt := resp.LastSamplePrompt()
	if prompt == "" {
		return fmt.Errorf("line %d: assertion failed: no sampling request received", i+1)
	}
	if !strings.Contains(prompt, parts[1]) {
		return fmt.Errorf("line %d: assertion failed: sampling prompt %q does not contain %q", i+1, prompt, parts[1])
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, fmt.Sprintf("sampling prompt contains %q", parts[1])))
	return nil
}
