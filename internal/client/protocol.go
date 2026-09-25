package client

import (
	"context"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
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

// RequestMeta is the per-request metadata (2026-07-28) for requests sent over
// the raw path, which the SDK does not fill in: protocol version, client info
// and the given client capabilities (nil means none).
func RequestMeta(session *mcp.ClientSession, capabilities map[string]any) map[string]any {
	if capabilities == nil {
		capabilities = map[string]any{}
	}
	return map[string]any{
		mcp.MetaKeyProtocolVersion:    session.InitializeResult().ProtocolVersion,
		mcp.MetaKeyClientInfo:         map[string]any{"name": version.AppName, "version": version.Version},
		mcp.MetaKeyClientCapabilities: capabilities,
	}
}
