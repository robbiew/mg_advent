// Package session tracks a caller's idle and total time limits.
//
// It holds no timers or goroutines: the door asks Next how long it may wait
// for the next key and reads input with that timeout, so expiry is handled on
// the main goroutine and normal cleanup (restoring the terminal, closing the
// BBS session) always runs.
package session

import "time"

// Reason identifies which limit ended a session.
type Reason int

const (
	// ReasonIdle means the caller pressed no key for the idle timeout.
	ReasonIdle Reason = iota + 1
	// ReasonTimeLimit means the session reached its total time allowance.
	ReasonTimeLimit
)

func (r Reason) String() string {
	switch r {
	case ReasonIdle:
		return "idle timeout"
	case ReasonTimeLimit:
		return "time limit"
	default:
		return "unknown"
	}
}

// Manager tracks one caller's session limits. It is not safe for concurrent use.
type Manager struct {
	idleTimeout  time.Duration
	deadline     time.Time
	lastActivity time.Time
	now          func() time.Time
}

// NewManager starts a session that ends after idleTimeout without a key, or
// after timeLimit in total, whichever comes first.
func NewManager(idleTimeout, timeLimit time.Duration) *Manager {
	return newManager(idleTimeout, timeLimit, time.Now)
}

func newManager(idleTimeout, timeLimit time.Duration, now func() time.Time) *Manager {
	start := now()
	return &Manager{
		idleTimeout:  idleTimeout,
		deadline:     start.Add(timeLimit),
		lastActivity: start,
		now:          now,
	}
}

// Touch records caller activity, restarting the idle timeout.
func (m *Manager) Touch() {
	m.lastActivity = m.now()
}

// Next returns how long the door may wait for the caller's next key and which
// limit ends that wait. A duration of zero or less means the session is over.
func (m *Manager) Next() (time.Duration, Reason) {
	now := m.now()
	idle := m.lastActivity.Add(m.idleTimeout).Sub(now)
	left := m.deadline.Sub(now)
	if left <= idle {
		return left, ReasonTimeLimit
	}
	return idle, ReasonIdle
}
