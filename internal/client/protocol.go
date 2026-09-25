package client

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StatelessRevision is the first protocol revision without initialize, ping
// and logging/setLevel (SEP-2575); the handshake is server/discover.
const StatelessRevision = "2026-07-28"

// IsStateless reports whether the session negotiated StatelessRevision or later.
func IsStateless(session *mcp.ClientSession) bool {
	return session.InitializeResult().ProtocolVersion >= StatelessRevision
}

// Ping checks that the server answers and returns the method that proved it.
// Since 2026-07-28 ping is removed; the server/discover exchange that opened
// the session is the liveness check then.
func Ping(ctx context.Context, session *mcp.ClientSession) (string, error) {
	if IsStateless(session) {
		return "server/discover", nil
	}
	return "ping", session.Ping(ctx, &mcp.PingParams{})
}

// WithLogLevel adds the per-request log level (2026-07-28) to meta.
func WithLogLevel(meta mcp.Meta, level string) mcp.Meta {
	if meta == nil {
		meta = mcp.Meta{}
	}
	meta[mcp.MetaKeyLogLevel] = level
	return meta
}
