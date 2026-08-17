package server

import (
	"sync"
	"testing"
	"time"
)

func TestLifecycleManagerGracePeriodAndShutdown(t *testing.T) {
	sm := NewStateManager("")
	var shutdownCalled bool
	var mu sync.Mutex

	// Configure short grace period (50ms) and short inactivity timeout (50ms)
	cfg := LifecycleConfig{
		StartupGracePeriod: 50 * time.Millisecond,
		InactivityTimeout:  50 * time.Millisecond,
	}

	lm := NewLifecycleManager(sm, cfg, func() {
		mu.Lock()
		shutdownCalled = true
		mu.Unlock()
	})
	defer lm.Stop()

	// 1. Immediately after creation, grace timer is ticking, activeClients == 0
	if lm.IsGraceExpired() {
		t.Errorf("expected graceExpired to be false initially")
	}

	// 2. Before grace period expires (at 20ms), shutdown should not have been called
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	if shutdownCalled {
		t.Errorf("shutdown called during startup grace period")
	}
	mu.Unlock()

	// 3. Wait for grace period to expire (total ~60ms) and inactivity timeout (total ~120ms)
	time.Sleep(120 * time.Millisecond)

	mu.Lock()
	if !shutdownCalled {
		t.Errorf("expected shutdown to be called after grace period and inactivity timeout")
	}
	mu.Unlock()
}

func TestLifecycleManagerClientConnectionCancelsIdle(t *testing.T) {
	sm := NewStateManager("")
	var shutdownCalled bool
	var mu sync.Mutex

	cfg := LifecycleConfig{
		StartupGracePeriod: 30 * time.Millisecond,
		InactivityTimeout:  60 * time.Millisecond,
	}

	lm := NewLifecycleManager(sm, cfg, func() {
		mu.Lock()
		shutdownCalled = true
		mu.Unlock()
	})
	defer lm.Stop()

	// Wait for grace period to expire
	time.Sleep(40 * time.Millisecond)
	if !lm.IsGraceExpired() {
		t.Errorf("expected grace period to be expired")
	}
	if !lm.IsIdleTimerRunning() {
		t.Errorf("expected idle countdown timer to be running")
	}

	// Client connects!
	lm.OnClientConnect()
	if lm.ActiveClients() != 1 {
		t.Errorf("expected 1 active client, got %d", lm.ActiveClients())
	}
	if lm.IsIdleTimerRunning() {
		t.Errorf("expected idle timer to be cancelled on client connect")
	}

	// Wait 80ms while client is connected
	time.Sleep(80 * time.Millisecond)
	mu.Lock()
	if shutdownCalled {
		t.Errorf("shutdown should not be called while a client is connected")
	}
	mu.Unlock()

	// Client disconnects
	lm.OnClientDisconnect()
	if lm.ActiveClients() != 0 {
		t.Errorf("expected 0 active clients, got %d", lm.ActiveClients())
	}
	if !lm.IsIdleTimerRunning() {
		t.Errorf("expected idle timer to start on client disconnect")
	}

	// Wait for inactivity timeout
	time.Sleep(80 * time.Millisecond)
	mu.Lock()
	if !shutdownCalled {
		t.Errorf("expected shutdown to be called after client disconnects and timeout elapses")
	}
	mu.Unlock()
}

func TestLifecycleManagerExecutionInhibitsShutdown(t *testing.T) {
	sm := NewStateManager("")
	// Set status to running
	sm.SetStatus(StatusRunning)

	var shutdownCalled bool
	var mu sync.Mutex

	cfg := LifecycleConfig{
		StartupGracePeriod: 30 * time.Millisecond,
		InactivityTimeout:  40 * time.Millisecond,
	}

	lm := NewLifecycleManager(sm, cfg, func() {
		mu.Lock()
		shutdownCalled = true
		mu.Unlock()
	})
	defer lm.Stop()

	// Wait for grace period + timeout (100ms)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if shutdownCalled {
		t.Errorf("shutdown should be inhibited while status is running")
	}
	mu.Unlock()

	// Finish execution
	sm.SetStatus(StatusCompleted)
	lm.OnExecutionStateChange()


	if !lm.IsIdleTimerRunning() {
		t.Errorf("expected idle timer to start after execution complete with 0 clients")
	}

	// Wait for inactivity timeout
	time.Sleep(60 * time.Millisecond)
	mu.Lock()
	if !shutdownCalled {
		t.Errorf("expected shutdown after execution completed and idle timeout elapsed")
	}
	mu.Unlock()
}
