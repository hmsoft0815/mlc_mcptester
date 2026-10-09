package mcptasks

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TaskParams are the params of tasks/get and tasks/cancel.
type TaskParams struct {
	mcp.ParamsBase
	TaskID string `json:"taskId"`
}

// UpdateParams are the params of tasks/update.
type UpdateParams struct {
	mcp.ParamsBase
	TaskID         string          `json:"taskId"`
	InputResponses json.RawMessage `json:"inputResponses"`
}

// TaskResult is a Task as returned by tools/call (resultType "task") and
// tasks/get (resultType "complete", with the status-specific fields).
type TaskResult struct {
	mcp.ResultBase
	ResultType     string              `json:"resultType"`
	TaskID         string              `json:"taskId"`
	Status         string              `json:"status"`
	StatusMessage  string              `json:"statusMessage,omitempty"`
	CreatedAt      string              `json:"createdAt"`
	LastUpdatedAt  string              `json:"lastUpdatedAt"`
	TTLMs          *int64              `json:"ttlMs"`
	PollIntervalMs int64               `json:"pollIntervalMs,omitempty"`
	Result         json.RawMessage     `json:"result,omitempty"`
	Error          *jsonrpc.Error      `json:"error,omitempty"`
	InputRequests  mcp.InputRequestMap `json:"inputRequests,omitempty"`
}

// AckResult is the empty acknowledgement of tasks/update and tasks/cancel.
type AckResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
}
