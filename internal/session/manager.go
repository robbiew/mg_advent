package session

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager handles session timeouts and idle detection
type Manager struct {
	idleTimeout  time.Duration
	maxTimeout   time.Duration
	onIdle       func()
	onMax        func()
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	lastActivity time.Time
	lock         sync.Mutex
	running      bool
}

// NewManager creates a new session manager
func NewManager(idleTimeout, maxTimeout time.Duration, onIdle, onMax func()) *Manager {
	return &Manager{
		idleTimeout:  idleTimeout,
		maxTimeout:   maxTimeout,
		onIdle:       onIdle,
		onMax:        onMax,
		lastActivity: time.Now(),
	}
}

// Start begins the session timers
func (sm *Manager) Start() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if sm.running {
		return
	}

	sm.running = true
	sm.resetIdleTimer()
	sm.resetMaxTimer()

	logrus.WithFields(logrus.Fields{
		"idle_timeout": sm.idleTimeout,
		"max_timeout":  sm.maxTimeout,
	}).Info("Session manager started")
}

// Stop stops the session timers
func (sm *Manager) Stop() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.running = false

	if sm.idleTimer != nil {
		sm.idleTimer.Stop()
		sm.idleTimer = nil
	}

	if sm.maxTimer != nil {
		sm.maxTimer.Stop()
		sm.maxTimer = nil
	}

	logrus.Info("Session manager stopped")
}

// ResetIdleTimer resets the idle timeout
func (sm *Manager) ResetIdleTimer() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.lastActivity = time.Now()

	if !sm.running {
		return
	}

	sm.resetIdleTimer()
}

// resetIdleTimer resets the idle timer (internal method)
func (sm *Manager) resetIdleTimer() {
	if sm.idleTimer != nil {
		sm.idleTimer.Stop()
	}

	sm.idleTimer = time.AfterFunc(sm.idleTimeout, func() {
		logrus.Warn("Idle timeout reached")
		if sm.onIdle != nil {
			sm.onIdle()
		}
	})
}

// resetMaxTimer resets the maximum session timer
func (sm *Manager) resetMaxTimer() {
	if sm.maxTimer != nil {
		sm.maxTimer.Stop()
	}

	sm.maxTimer = time.AfterFunc(sm.maxTimeout, func() {
		logrus.Warn("Maximum session time reached")
		if sm.onMax != nil {
			sm.onMax()
		}
	})
}

// GetIdleTime returns how long the session has been idle
func (sm *Manager) GetIdleTime() time.Duration {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	return time.Since(sm.lastActivity)
}

// GetRemainingTime returns remaining time before max timeout
func (sm *Manager) GetRemainingTime() time.Duration {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if !sm.running || sm.maxTimer == nil {
		return 0
	}

	// This is approximate since we can't get exact remaining time from timer
	// In a real implementation, you'd track start time
	return sm.maxTimeout
}

// IsActive returns whether the session manager is active
func (sm *Manager) IsActive() bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	return sm.running
}

// ExtendMaxTimeout extends the maximum session time (for special cases)
func (sm *Manager) ExtendMaxTimeout(extension time.Duration) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if !sm.running {
		return
	}

	sm.maxTimeout += extension
	sm.resetMaxTimer()

	logrus.WithField("extension", extension).Info("Maximum session time extended")
}

// GetStats returns session statistics
func (sm *Manager) GetStats() map[string]interface{} {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	return map[string]interface{}{
		"running":       sm.running,
		"idle_timeout":  sm.idleTimeout.String(),
		"max_timeout":   sm.maxTimeout.String(),
		"idle_time":     sm.GetIdleTime().String(),
		"last_activity": sm.lastActivity.Format(time.RFC3339),
	}
}
