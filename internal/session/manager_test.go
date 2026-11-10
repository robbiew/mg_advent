package session

import (
	"testing"
	"time"
)

func TestNextPicksEarliestLimit(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	m := newManager(5*time.Minute, 42*time.Minute, clock)

	if wait, reason := m.Next(); wait != 5*time.Minute || reason != ReasonIdle {
		t.Fatalf("fresh session: %v %v", wait, reason)
	}

	now = now.Add(4 * time.Minute)
	m.Touch()
	if wait, reason := m.Next(); wait != 5*time.Minute || reason != ReasonIdle {
		t.Fatalf("after activity: %v %v", wait, reason)
	}

	now = now.Add(36 * time.Minute) // 40 min in; touched at 4 → idle expired
	if wait, reason := m.Next(); wait > 0 || reason != ReasonIdle {
		t.Fatalf("idle should have expired: %v %v", wait, reason)
	}

	m.Touch() // 2 minutes of BBS time left
	if wait, reason := m.Next(); wait != 2*time.Minute || reason != ReasonTimeLimit {
		t.Fatalf("time limit should win: %v %v", wait, reason)
	}
}
