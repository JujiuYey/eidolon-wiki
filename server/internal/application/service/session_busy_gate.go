package service

import (
	"errors"
	"strings"
	"sync"
)

// ErrSessionRewindLocked reports that a rewind is currently in progress on
// the session, so a new send turn must be rejected. The HTTP layer maps it
// to 409 Conflict.
var ErrSessionRewindLocked = errors.New("session rewind: a rewind is in progress")

// SessionBusyGate is process-local exclusion between send and rewind.
// weknora-slim runs single-node, so this map is the only turn gate.
type SessionBusyGate struct {
	mu     sync.Mutex
	states map[string]*sessionBusyState
}

type sessionBusyState struct {
	sendRefs int
	rewind   bool
}

// NewSessionBusyGate returns an empty gate. Container provides one shared
// instance so send and rewind see each other in-process.
func NewSessionBusyGate() *SessionBusyGate {
	return &SessionBusyGate{states: make(map[string]*sessionBusyState)}
}

func (g *SessionBusyGate) stateLocked(sessionID string) *sessionBusyState {
	st := g.states[sessionID]
	if st == nil {
		st = &sessionBusyState{}
		g.states[sessionID] = st
	}
	return st
}

func (g *SessionBusyGate) dropLocked(sessionID string, st *sessionBusyState) {
	if st.sendRefs <= 0 && !st.rewind {
		delete(g.states, sessionID)
	}
}

// HoldSend marks sessionID as generating. Fails when rewind already holds it.
func (g *SessionBusyGate) HoldSend(sessionID string) (func(), error) {
	noop := func() {}
	if g == nil {
		return noop, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return noop, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	st := g.stateLocked(sessionID)
	if st.rewind {
		return noop, ErrSessionRewindLocked
	}
	st.sendRefs++
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			defer g.mu.Unlock()
			st := g.states[sessionID]
			if st == nil {
				return
			}
			st.sendRefs--
			if st.sendRefs < 0 {
				st.sendRefs = 0
			}
			g.dropLocked(sessionID, st)
		})
	}, nil
}

// TryLockRewind takes exclusive rewind ownership. Fails when a send hold
// or another rewind already owns the session.
func (g *SessionBusyGate) TryLockRewind(sessionID string) (func(), error) {
	noop := func() {}
	if g == nil {
		return noop, nil
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return noop, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	st := g.stateLocked(sessionID)
	if st.rewind || st.sendRefs > 0 {
		return noop, ErrRewindSourceBusy
	}
	st.rewind = true
	var once sync.Once
	return func() {
		once.Do(func() {
			g.mu.Lock()
			defer g.mu.Unlock()
			st := g.states[sessionID]
			if st == nil {
				return
			}
			st.rewind = false
			g.dropLocked(sessionID, st)
		})
	}, nil
}

// RewindHeld reports whether rewind currently owns sessionID.
func (g *SessionBusyGate) RewindHeld(sessionID string) bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	st := g.states[strings.TrimSpace(sessionID)]
	return st != nil && st.rewind
}
