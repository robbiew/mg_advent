package main

import (
	"sync"
	"time"
)

type TimerManager struct {
	idleTimer    *time.Timer
	maxTimer     *time.Timer
	idleDuration time.Duration
	maxDuration  time.Duration
	onIdle       func() // Callback for idle timeout
	onMax        func() // Callback for max timeout
	lock         sync.Mutex
}

// NewTimerManager creates a new TimerManager
func NewTimerManager(idleDuration, maxDuration time.Duration, onIdle, onMax func()) *TimerManager {
	return &TimerManager{
		idleDuration: idleDuration,
		maxDuration:  maxDuration,
		onIdle:       onIdle,
		onMax:        onMax,
	}
}

// StartTimers initializes both timers
func (tm *TimerManager) StartTimers() {
	tm.ResetIdleTimer()
	tm.ResetMaxTimer()
}

// ResetIdleTimer resets the idle timer
func (tm *TimerManager) ResetIdleTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.idleTimer != nil {
		tm.idleTimer.Stop()
	}
	tm.idleTimer = time.AfterFunc(tm.idleDuration, tm.onIdle)
}

// ResetMaxTimer resets the maximum duration timer
func (tm *TimerManager) ResetMaxTimer() {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	if tm.maxTimer != nil {
		tm.maxTimer.Stop()
	}
	tm.maxTimer = time.AfterFunc(tm.maxDuration, tm.onMax)
}
