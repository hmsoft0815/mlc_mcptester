package mcptasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// OwnerFunc returns the identity a request acts for; a task is bound to the
// identity that created it, and tasks/get, tasks/update and tasks/cancel of
// another identity find no task (-32602), as for an unknown id. "" means no
// identity: such tasks are protected by their unguessable id only.
type OwnerFunc func(extra *mcp.RequestExtra) string

// DefaultOwner is the user id of the verified bearer token or, if the
// verifier sets none, a digest of the Authorization header — then a task is
// bound to the token, and a refreshed token no longer reaches it. Set
// Store.Owner when the verifier knows a stable identity it does not put
// into TokenInfo.UserID.
func DefaultOwner(extra *mcp.RequestExtra) string {
	if extra == nil {
		return ""
	}
	if ti := extra.TokenInfo; ti != nil && ti.UserID != "" {
		return "user:" + ti.UserID
	}
	if a := extra.Header.Get("Authorization"); a != "" {
		sum := sha256.Sum256([]byte(a))
		return "token:" + hex.EncodeToString(sum[:])
	}
	return ""
}

// ownerKey carries the identity of a tasks/* request from the middleware,
// which sees the request extra, to the method handlers, which do not.
type ownerKey struct{}

func (s *Store) ownerOf(req mcp.Request) string {
	owner := s.Owner
	if owner == nil {
		owner = DefaultOwner
	}
	return owner(req.GetExtra())
}

func ownerFrom(ctx context.Context) string {
	owner, _ := ctx.Value(ownerKey{}).(string)
	return owner
}
