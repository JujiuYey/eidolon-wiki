package session

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// completionEventRecorder is the StreamManager stub shared by the completion
// tests: it records appended events and nothing else.
type completionEventRecorder struct {
	interfaces.StreamManager
	events []interfaces.StreamEvent
}

func (s *completionEventRecorder) AppendEvent(_ context.Context, _, _ string, e interfaces.StreamEvent) error {
	s.events = append(s.events, e)
	return nil
}
