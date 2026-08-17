package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/internal/elevation"
	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

type serverObserver struct {
	sm      *StateManager
	runID   string
	mu      sync.Mutex
	eventID int64
}

func (o *serverObserver) nextEventID() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.eventID++
	return o.eventID
}

func (o *serverObserver) OnRunStart(runID string, pb *playbook.Playbook) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.RunID = runID
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnSectionStart(section playbook.Section, index int, total int) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	msg := fmt.Sprintf("Processing Section %d/%d: %s", index, total, section.Title)
	o.sm.snapshot.Logs = append(o.sm.snapshot.Logs, msg)
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnAssertionStart(assertion playbook.Assertion, index int, total int) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.ActiveAssertionCode = assertion.Code
	for i := range o.sm.snapshot.Assertions {
		if o.sm.snapshot.Assertions[i].Code == assertion.Code {
			o.sm.snapshot.Assertions[i].Status = "running"
			break
		}
	}
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnAssertionComplete(assertion playbook.Assertion, result director.AssertionProgressResult) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()

	for i := range o.sm.snapshot.Assertions {
		if o.sm.snapshot.Assertions[i].Code == assertion.Code {
			o.sm.snapshot.Assertions[i].Status = result.Status
			o.sm.snapshot.Assertions[i].Passed = result.Passed
			o.sm.snapshot.Assertions[i].Score = result.Score
			o.sm.snapshot.Assertions[i].MinScore = result.MinScore
			o.sm.snapshot.Assertions[i].DurationMs = result.DurationMs
			o.sm.snapshot.Assertions[i].Output = result.Output
			break
		}
	}
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnLog(message string) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.Logs = append(o.sm.snapshot.Logs, message)
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnRunComplete(trace executor.ExecutionTrace, rep report.FinalReport) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.ActiveAssertionCode = ""
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnRunCancelled(runID string, partialTrace executor.ExecutionTrace) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.ActiveAssertionCode = ""
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (o *serverObserver) OnRunError(err error) {
	o.sm.mu.Lock()
	defer o.sm.mu.Unlock()
	o.sm.snapshot.Logs = append(o.sm.snapshot.Logs, fmt.Sprintf("Error: %v", err))
	o.sm.snapshot.LastEventID = o.nextEventID()
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.state.mu.Lock()
	// Check if playbook is loaded
	if s.state.playbook == nil {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_PLAYBOOK",
				Message: "No playbook loaded. Load a playbook before starting execution.",
			},
		})
		return
	}

	// Check if validation errors exist
	for _, err := range s.state.errors {
		if err.Code == ErrCodePlaybookValidationFailed {
			s.state.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": AppError{
					Code:    ErrCodePlaybookValidationFailed,
					Message: "Cannot run playbook with validation errors",
					Detail:  err.Detail,
				},
			})
			return
		}
	}

	// Check status
	if s.state.status == StatusRunning ||
		s.state.status == StatusRunningElevating ||
		s.state.status == StatusRunningCancelling ||
		s.state.status == StatusCompletedSubmitting {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Execution is already in progress",
			},
		})
		return
	}

	pb := s.state.playbook
	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	s.state.activeRunID = runID
	s.state.activeRunCancel = cancel
	s.state.activeRunDone = make(chan struct{})
	s.state.isCancelled = false
	s.state.runStartTime = time.Now()
	s.state.lastReport = nil
	s.state.errors = make([]AppError, 0)

	// Reset snapshot assertions
	for i := range s.state.snapshot.Assertions {
		s.state.snapshot.Assertions[i].Status = "pending"
		s.state.snapshot.Assertions[i].Passed = false
		s.state.snapshot.Assertions[i].Score = 0
		s.state.snapshot.Assertions[i].DurationMs = 0
		s.state.snapshot.Assertions[i].Output = ""
	}
	s.state.snapshot.Logs = make([]string, 0)
	s.state.snapshot.ActiveAssertionCode = ""

	if pb.RequiresElevation() {
		s.state.status = StatusRunningElevating
	} else {
		s.state.status = StatusRunning
	}
	s.state.snapshot.Status = s.state.status

	obs := &serverObserver{
		sm:    s.state,
		runID: runID,
	}

	runDoneChan := s.state.activeRunDone
	s.state.mu.Unlock()

	// Launch execution in background goroutine
	go func() {
		defer close(runDoneChan)

		// 1. Setup Elevation if required
		var cleanupElevation func() = func() {}
		if pb.RequiresElevation() {
			var err error
			cleanupElevation, err = elevation.SetupElevation(pb)
			if err != nil {
				s.state.mu.Lock()
				s.state.status = StatusLoaded
				s.state.errors = append(s.state.errors, AppError{
					Code:    ErrCodeElevationFailed,
					Message: fmt.Sprintf("Elevation setup failed: %v", err),
				})
				s.state.snapshot.Status = StatusLoaded
				s.state.mu.Unlock()
				return
			}
			s.state.mu.Lock()
			if s.state.status == StatusRunningElevating {
				s.state.status = StatusRunning
				s.state.snapshot.Status = StatusRunning
			}
			s.state.mu.Unlock()
		}
		defer cleanupElevation()

		// 2. Execute Playbook
		trace := director.RunWithObserver(ctx, *pb, obs, runID)
		res := report.GenerateReport(trace)

		s.state.mu.Lock()
		defer s.state.mu.Unlock()

		s.state.lastReport = &res
		s.state.snapshot.DurationMs = time.Since(s.state.runStartTime).Milliseconds()

		if ctx.Err() != nil {
			// Cancelled run
			s.state.isCancelled = true
			s.state.status = StatusLoaded
			s.state.errors = append(s.state.errors, AppError{
				Code:    ErrCodeExecutionAborted,
				Message: "Execution cancelled by user",
			})
			s.state.snapshot.Status = "cancelled"
			return
		}

		// 3. Execution Completed - Synchronous Local Folder Persistence
		if s.state.reportDestination.FolderSource != FolderSourceOff {
			folderDir := s.state.reportDestination.Folder
			if folderDir == "" {
				folderDir = "reports"
			}
			if err := reportwriter.WriteToFolder(folderDir, res); err != nil {
				s.state.errors = append(s.state.errors, AppError{
					Code:    ErrCodeFolderWriteFailed,
					Message: fmt.Sprintf("Failed to write report to folder: %v", err),
				})
			}
		}

		// 4. Destination status transition
		hasHTTPS := (s.state.reportDestination.HttpsSource == HttpsSourcePlaybook && pb.ReportDestinationHTTPS != nil) ||
			(s.state.reportDestination.HttpsSource == HttpsSourceCustom && s.state.reportDestination.HTTPS != nil && s.state.reportDestination.HTTPS.URL != "")

		if hasHTTPS {
			s.state.status = StatusCompletedConfirmingSubmission
		} else {
			s.state.status = StatusCompleted
		}
		s.state.snapshot.Status = s.state.status
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"run_id": runID,
	})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.state.mu.Lock()
	if s.state.status != StatusRunning &&
		s.state.status != StatusRunningElevating &&
		s.state.status != StatusRunningCancelling {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "No active execution to cancel",
			},
		})
		return
	}

	if s.state.status == StatusRunningCancelling {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.state.GetStateResponse())
		return
	}

	s.state.status = StatusRunningCancelling
	s.state.snapshot.Status = StatusRunningCancelling
	cancel := s.state.activeRunCancel
	doneChan := s.state.activeRunDone
	s.state.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if doneChan != nil {
		select {
		case <-doneChan:
		case <-time.After(10 * time.Second):
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.state.GetStateResponse())
}

func (s *Server) handleExecutionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := s.state.GetExecutionSnapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}
