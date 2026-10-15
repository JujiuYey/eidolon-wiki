// Package service - in-place session rewind.
//
// Rewind truncates the current session at a chosen user or assistant message.
// The sandbox subsystem that carried per-turn git checkpoints was removed from
// this deployment, so rewind is conversation-only: there is no workspace to
// roll back, and the result always reports NO_SANDBOX as the reason.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

// RewindSkipReason explains why rewind left the workspace alone. The
// conversation is still truncated; the code is so the client can say so.
type RewindSkipReason string

const (
	// RewindSkipNoSandbox means the deployment has no sandbox subsystem, so
	// rewind truncates the conversation only. This is the only reason this
	// deployment ever reports.
	RewindSkipNoSandbox RewindSkipReason = "NO_SANDBOX"
)

var (
	// ErrRewindSourceBusy reports that the session has an agent turn in
	// flight, or the rewind point is an unfinished assistant message. HTTP 409.
	ErrRewindSourceBusy = errors.New("session rewind: session has an active turn")

	// ErrRewindSessionNotFound covers a missing session and a session the
	// caller does not own. Both map to HTTP 404 so ownership is not enumerable.
	ErrRewindSessionNotFound = errors.New("session rewind: session not found")

	// ErrRewindMessageNotFound covers a missing message and a message that
	// belongs to a different session. HTTP 404.
	ErrRewindMessageNotFound = errors.New("session rewind: rewind point message not found")

	// ErrRewindMessageRole is returned when the rewind point exists but is
	// neither a user nor an assistant message. HTTP 400.
	ErrRewindMessageRole = errors.New("session rewind: rewind point must be a user or assistant message")
)

const (
	rewindIncompleteLookback = 512
	rewindDeleteAttempts     = 3
	rewindDeleteRetryWait    = 50 * time.Millisecond
)

func isRewindNotFound(err error) bool {
	return errors.Is(err, apperrors.ErrSessionNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}

// RewindResult is what the HTTP layer renders. WorkspaceReset is always
// false in this deployment: there is no sandbox workspace to reset.
type RewindResult struct {
	DeletedMessages int              `json:"deleted_messages"`
	WorkspaceReset  bool             `json:"workspace_reset"`
	Reason          RewindSkipReason `json:"reason"`
}

type rewindSessionStore interface {
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.Session, error)
}

type rewindMessageStore interface {
	GetMessage(ctx context.Context, sessionID, messageID string) (*types.Message, error)
	GetRecentMessagesBySession(ctx context.Context, sessionID string, limit int) ([]*types.Message, error)
	DeleteMessagesFrom(
		ctx context.Context, sessionID string, boundary time.Time, boundaryID string, inclusive bool,
	) ([]*types.Message, error)
}

type rewindIncompleteFinder interface {
	SessionHasIncompleteAssistant(ctx context.Context, sessionID string) (bool, error)
}

type rewindArtifactJanitor interface {
	ListLiveArtifactsByMessageIDs(
		ctx context.Context, sessionID string, messageIDs []string,
	) ([]types.MessageArtifactRecord, error)
	SoftDeleteSessionArtifacts(
		ctx context.Context, sessionID string, refs []types.ArtifactRef, at time.Time,
	) ([]types.ArtifactRef, error)
}

type rewindLiveRunReader interface {
	GetLiveRun(ctx context.Context, sessionID string) (assistantMessageID, requestID string, err error)
}

type rewindStreamDropper interface {
	DropMessageStreams(ctx context.Context, sessionID string, messageIDs []string) error
}

type rewindKnowledgeCleaner interface {
	DeleteMessageKnowledge(ctx context.Context, knowledgeID string)
}

type rewindSuggestionCleaner interface {
	DeleteByMessageID(ctx context.Context, tenantID uint64, sessionID, messageID string) error
}

// rewindSuggestionBatchCleaner is the one-statement form of the above.
type rewindSuggestionBatchCleaner interface {
	DeleteByMessageIDs(ctx context.Context, tenantID uint64, sessionID string, messageIDs []string) error
}

// SessionRewindService implements in-place rewind.
type SessionRewindService struct {
	sessions    rewindSessionStore
	messages    rewindMessageStore
	knowledge   rewindKnowledgeCleaner
	suggestions rewindSuggestionCleaner
	liveRuns    rewindLiveRunReader
	busyGate    *SessionBusyGate
}

// NewSessionRewindService wires the service.
func NewSessionRewindService(
	sessions rewindSessionStore,
	messages rewindMessageStore,
	knowledge rewindKnowledgeCleaner,
	suggestions rewindSuggestionCleaner,
) *SessionRewindService {
	return &SessionRewindService{
		sessions:    sessions,
		messages:    messages,
		knowledge:   knowledge,
		suggestions: suggestions,
		busyGate:    NewSessionBusyGate(),
	}
}

// NewSessionRewindServiceFromRepos is the DI-friendly constructor.
func NewSessionRewindServiceFromRepos(
	sessions interfaces.SessionRepository,
	messages interfaces.MessageRepository,
	knowledge interfaces.MessageService,
	suggestions interfaces.MessageSuggestionRepository,
	streams interfaces.StreamManager,
	busyGate *SessionBusyGate,
) *SessionRewindService {
	s := NewSessionRewindService(sessions, messages, knowledge, suggestions)
	s.liveRuns = streams
	if busyGate != nil {
		s.busyGate = busyGate
	}
	return s
}

// Rewind truncates sessionID at messageID.
//
// Errors are reserved for conditions the user must act on: a bad request, a
// session they do not own, or a busy source.
func (s *SessionRewindService) Rewind(
	ctx context.Context,
	tenantID uint64,
	userID string,
	sessionID string,
	messageID string,
) (*RewindResult, error) {
	source, err := s.sessions.GetByID(ctx, tenantID, sessionID)
	if err != nil {
		if isRewindNotFound(err) {
			return nil, ErrRewindSessionNotFound
		}
		return nil, fmt.Errorf("session rewind: load session: %w", err)
	}
	if source == nil {
		return nil, ErrRewindSessionNotFound
	}
	// Rewind is a write. Empty user_id rows are tenant-level channel/legacy
	// sessions: list/read scope still surfaces them, but mutating history
	// requires an exact owner match.
	if userID == "" || source.UserID != userID {
		return nil, ErrRewindSessionNotFound
	}

	rewindPoint, err := s.messages.GetMessage(ctx, sessionID, messageID)
	if err != nil {
		if isRewindNotFound(err) {
			return nil, ErrRewindMessageNotFound
		}
		return nil, fmt.Errorf("session rewind: load rewind point: %w", err)
	}
	if rewindPoint == nil {
		return nil, ErrRewindMessageNotFound
	}
	if rewindPoint.Role != "user" && rewindPoint.Role != "assistant" {
		return nil, ErrRewindMessageRole
	}
	if rewindPoint.Role == "assistant" && !rewindPoint.IsCompleted {
		return nil, ErrRewindSourceBusy
	}

	if err := s.rejectIfBusy(ctx, sessionID); err != nil {
		return nil, err
	}

	// Client abort must not leave a half-applied truncate. Once we are past
	// the cheap busy reject, finish the delete even if the HTTP request is
	// gone.
	persistCtx := context.WithoutCancel(ctx)
	unlock, err := s.lockRewind(persistCtx, sessionID)
	if err != nil {
		return nil, err
	}
	defer unlock()

	inclusive := rewindPoint.Role == "user"
	deleted, err := s.deleteMessagesFrom(
		persistCtx, sessionID, rewindPoint.CreatedAt, rewindPoint.ID, inclusive,
	)
	if err != nil {
		return nil, fmt.Errorf("session rewind: delete messages: %w", err)
	}

	s.cleanupDeleted(persistCtx, source.TenantID, sessionID, deleted)

	logger.Infof(ctx,
		"[SessionRewind] session=%s rewind_point=%s deleted=%d",
		sessionID, rewindPoint.ID, len(deleted))

	return &RewindResult{
		DeletedMessages: len(deleted),
		WorkspaceReset:  false,
		Reason:          RewindSkipNoSandbox,
	}, nil
}

func (s *SessionRewindService) deleteMessagesFrom(
	ctx context.Context, sessionID string, boundary time.Time, boundaryID string, inclusive bool,
) ([]*types.Message, error) {
	if s == nil || s.messages == nil {
		return nil, nil
	}
	var lastErr error
	for attempt := 1; attempt <= rewindDeleteAttempts; attempt++ {
		deleted, err := s.messages.DeleteMessagesFrom(
			ctx, sessionID, boundary, boundaryID, inclusive,
		)
		if err == nil {
			return deleted, nil
		}
		lastErr = err
		if attempt == rewindDeleteAttempts {
			break
		}
		timer := time.NewTimer(rewindDeleteRetryWait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, err
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func (s *SessionRewindService) lockRewind(_ context.Context, sessionID string) (func(), error) {
	unlock, err := s.busyGate.TryLockRewind(sessionID)
	if err != nil {
		return nil, ErrRewindSourceBusy
	}
	return unlock, nil
}

func (s *SessionRewindService) rejectIfBusy(ctx context.Context, sessionID string) error {
	if s.liveRuns != nil {
		liveID, _, err := s.liveRuns.GetLiveRun(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("session rewind: check live run: %w", err)
		}
		if strings.TrimSpace(liveID) != "" {
			return ErrRewindSourceBusy
		}
	}
	return s.rejectIfIncompleteTurn(ctx, sessionID)
}

func (s *SessionRewindService) rejectIfIncompleteTurn(ctx context.Context, sessionID string) error {
	if s.messages == nil {
		return nil
	}
	if finder, ok := s.messages.(rewindIncompleteFinder); ok {
		busy, err := finder.SessionHasIncompleteAssistant(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("session rewind: check incomplete turn: %w", err)
		}
		if busy {
			return ErrRewindSourceBusy
		}
		return nil
	}
	msgs, err := s.messages.GetRecentMessagesBySession(ctx, sessionID, rewindIncompleteLookback)
	if err != nil {
		return fmt.Errorf("session rewind: check incomplete turn: %w", err)
	}
	for _, msg := range msgs {
		if msg != nil && msg.Role == "assistant" && !msg.IsCompleted {
			return ErrRewindSourceBusy
		}
	}
	return nil
}

// deleteSuggestions drops the follow-up questions hanging off the deleted
// messages, in one statement when the store can batch. A long rewind deletes
// hundreds of messages and the rewind lock is held for all of it.
func (s *SessionRewindService) deleteSuggestions(
	ctx context.Context, tenantID uint64, sessionID string, ids []string,
) {
	if s.suggestions == nil {
		return
	}
	if batch, ok := s.suggestions.(rewindSuggestionBatchCleaner); ok {
		if err := batch.DeleteByMessageIDs(ctx, tenantID, sessionID, ids); err != nil {
			logger.Warnf(ctx, "[SessionRewind] delete suggestions for session %s: %v", sessionID, err)
		}
		return
	}
	for _, id := range ids {
		if err := s.suggestions.DeleteByMessageID(ctx, tenantID, sessionID, id); err != nil {
			logger.Warnf(ctx, "[SessionRewind] delete suggestions for message %s: %v", id, err)
		}
	}
}

func (s *SessionRewindService) cleanupDeleted(
	ctx context.Context, tenantID uint64, sessionID string, deleted []*types.Message,
) {
	ids := make([]string, 0, len(deleted))
	for _, msg := range deleted {
		if msg == nil || msg.ID == "" {
			continue
		}
		ids = append(ids, msg.ID)
		if s.knowledge != nil && strings.TrimSpace(msg.KnowledgeID) != "" {
			s.knowledge.DeleteMessageKnowledge(ctx, msg.KnowledgeID)
		}
	}
	if len(ids) == 0 {
		return
	}
	s.deleteSuggestions(ctx, tenantID, sessionID, ids)
	if dropper, ok := s.liveRuns.(rewindStreamDropper); ok {
		if err := dropper.DropMessageStreams(ctx, sessionID, ids); err != nil {
			logger.Warnf(ctx, "[SessionRewind] drop streams for session %s: %v", sessionID, err)
		}
	}
	janitor, ok := s.messages.(rewindArtifactJanitor)
	if !ok {
		return
	}
	records, err := janitor.ListLiveArtifactsByMessageIDs(ctx, sessionID, ids)
	if err != nil {
		logger.Warnf(ctx, "[SessionRewind] list artifacts for session %s: %v", sessionID, err)
		return
	}
	refs := make([]types.ArtifactRef, 0, len(records))
	for _, row := range records {
		if row.MessageID == "" || row.Position < 0 {
			continue
		}
		refs = append(refs, types.ArtifactRef{MessageID: row.MessageID, Position: row.Position})
	}
	if len(refs) == 0 {
		return
	}
	if _, err := janitor.SoftDeleteSessionArtifacts(ctx, sessionID, refs, time.Now()); err != nil {
		logger.Warnf(ctx, "[SessionRewind] tombstone artifacts for session %s: %v", sessionID, err)
	}
}
