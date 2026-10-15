package access

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// MessageFileLookup is the narrow message-service surface needed by the
// message-scoped file proxy. Keeping it small makes the authorization boundary
// independently testable.
type MessageFileLookup interface {
	GetMessage(ctx context.Context, sessionID, messageID string) (*types.Message, error)
}
