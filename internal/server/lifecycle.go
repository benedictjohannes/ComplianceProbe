package server

import (
	"sync"
	"time"
)

// LifecycleConfig configures the auto-shutdown lifecycle policy.
type LifecycleConfig struct {
	StartupGracePeriod   time.Duration
	InactivityTimeout    time.Duration
	DisableAutoShutdown  bool
}

// LifecycleManager manages startup grace periods, active SSE connections, and auto-shutdown timers.
type LifecycleManager struct {
	mu                 sync.Mutex
	startupGracePeriod time.Duration
	inactivityTimeout  time.Duration
	disabled           bool

	activeClients int
	graceTimer    *time.Timer
	idleTimer     *time.Timer
	graceExpired  bool

	state      *StateManager
	onShutdown func()
	stopped    bool
}

// NewLifecycleManager creates and initializes a LifecycleManager.
func NewLifecycleManager(state *StateManager, cfg LifecycleConfig, onShutdown func()) *LifecycleManager {
	gracePeriod := cfg.StartupGracePeriod
	if gracePeriod <= 0 {
		gracePeriod = 5 * time.Minute
	}

	idleTimeout := cfg.InactivityTimeout
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}

	lm := &LifecycleManager{
		startupGracePeriod: gracePeriod,
		inactivityTimeout:  idleTimeout,
		disabled:           cfg.DisableAutoShutdown,
		state:              state,
		onShutdown:         onShutdown,
	}

	if !lm.disabled {
		lm.graceTimer = time.AfterFunc(gracePeriod, lm.onGracePeriodExpired)
	} else {
		lm.graceExpired = true
	}

	return lm
}

// OnClientConnect is called whenever a new SSE client connects.
func (lm *LifecycleManager) OnClientConnect() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.activeClients++
	if lm.idleTimer != nil {
		lm.idleTimer.Stop()
		lm.idleTimer = nil
	}
}

// OnClientDisconnect is called whenever an SSE client disconnects.
func (lm *LifecycleManager) OnClientDisconnect() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.activeClients--
	if lm.activeClients < 0 {
		lm.activeClients = 0
	}

	if lm.activeClients == 0 && lm.graceExpired && !lm.isExecutionActiveLocked() && !lm.stopped && !lm.disabled {
		if lm.idleTimer != nil {
			lm.idleTimer.Stop()
		}
		lm.idleTimer = time.AfterFunc(lm.inactivityTimeout, lm.onIdleTimeoutExpired)
	}
}

// OnExecutionStateChange is called whenever execution completes or transitions to check if idle countdown should start.
func (lm *LifecycleManager) OnExecutionStateChange() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.activeClients == 0 && lm.graceExpired && !lm.isExecutionActiveLocked() && !lm.stopped && !lm.disabled {
		if lm.idleTimer == nil {
			lm.idleTimer = time.AfterFunc(lm.inactivityTimeout, lm.onIdleTimeoutExpired)
		}
	}
}

func (lm *LifecycleManager) onGracePeriodExpired() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.graceExpired = true
	if lm.activeClients == 0 && !lm.isExecutionActiveLocked() && !lm.stopped && !lm.disabled {
		if lm.idleTimer == nil {
			lm.idleTimer = time.AfterFunc(lm.inactivityTimeout, lm.onIdleTimeoutExpired)
		}
	}
}

func (lm *LifecycleManager) onIdleTimeoutExpired() {
	lm.mu.Lock()
	if lm.stopped || lm.disabled || lm.activeClients > 0 || lm.isExecutionActiveLocked() {
		lm.idleTimer = nil
		lm.mu.Unlock()
		return
	}
	lm.idleTimer = nil
	shutdownFunc := lm.onShutdown
	lm.mu.Unlock()

	if shutdownFunc != nil {
		shutdownFunc()
	}
}

func (lm *LifecycleManager) isExecutionActiveLocked() bool {
	if lm.state == nil {
		return false
	}
	status := lm.state.GetStatus()
	return status == StatusRunningElevating ||
		status == StatusRunning ||
		status == StatusRunningCancelling ||
		status == StatusCompletedSubmitting
}

// Stop terminates all background timers.
func (lm *LifecycleManager) Stop() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.stopped = true
	if lm.graceTimer != nil {
		lm.graceTimer.Stop()
		lm.graceTimer = nil
	}
	if lm.idleTimer != nil {
		lm.idleTimer.Stop()
		lm.idleTimer = nil
	}
}

// ActiveClients returns the count of active SSE client connections.
func (lm *LifecycleManager) ActiveClients() int {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.activeClients
}

// IsGraceExpired returns true if the initial startup grace period has expired.
func (lm *LifecycleManager) IsGraceExpired() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.graceExpired
}

// IsIdleTimerRunning returns true if the inactivity countdown timer is active.
func (lm *LifecycleManager) IsIdleTimerRunning() bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	return lm.idleTimer != nil
}
