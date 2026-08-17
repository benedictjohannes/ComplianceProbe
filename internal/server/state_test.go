package server

import (
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
)

func TestStateManager(t *testing.T) {
	sm := NewStateManager("cli-reports-folder")

	// Initial status
	if sm.GetStatus() != StatusIdle {
		t.Errorf("expected initial status %s, got %s", StatusIdle, sm.GetStatus())
	}
	if sm.HasValidationErrors() {
		t.Errorf("expected HasValidationErrors() to be false initially")
	}
	if sm.GetPlaybook() != nil {
		t.Errorf("expected GetPlaybook() to be nil initially")
	}
	rep, cancelled := sm.GetLastReport()
	if rep != nil || cancelled {
		t.Errorf("expected GetLastReport() to be nil initially, got rep=%v, cancelled=%v", rep, cancelled)
	}
	if !sm.CanMutate() {
		t.Errorf("expected CanMutate() to be true initially")
	}

	// Set valid playbook
	pb := &playbook.Playbook{
		Title: "Test Playbook",
		Sections: []playbook.Section{
			{
				Title: "Sec 1",
				Assertions: []playbook.Assertion{
					{Code: "A1", Title: "Ass 1"},
				},
			},
		},
	}
	sm.SetPlaybook(pb, []byte("title: Test Playbook"), nil)

	if sm.GetStatus() != StatusLoaded {
		t.Errorf("expected status %s after SetPlaybook, got %s", StatusLoaded, sm.GetStatus())
	}
	if sm.GetPlaybook() != pb {
		t.Errorf("expected GetPlaybook() to return set playbook")
	}
	if sm.HasValidationErrors() {
		t.Errorf("expected HasValidationErrors() to be false with nil valErrors")
	}

	// Playbook inspection
	insp, err := sm.GetPlaybookInspection()
	if err != nil || insp == nil || insp.Title != "Test Playbook" {
		t.Errorf("expected inspection title 'Test Playbook', got err=%v, insp=%+v", err, insp)
	}

	// Set status
	sm.SetStatus(StatusRunning)
	if sm.GetStatus() != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, sm.GetStatus())
	}
	if sm.CanMutate() {
		t.Errorf("expected CanMutate() to be false during running")
	}

	// Unload playbook while running (fails mutation)
	if err := sm.UnloadPlaybook(); err == nil {
		t.Errorf("expected UnloadPlaybook() to return error when running")
	}

	// Set status back to loaded
	sm.SetStatus(StatusLoaded)
	if err := sm.UnloadPlaybook(); err != nil {
		t.Errorf("expected UnloadPlaybook() to succeed when loaded, got err=%v", err)
	}
	if sm.GetStatus() != StatusIdle {
		t.Errorf("expected status %s after unload, got %s", StatusIdle, sm.GetStatus())
	}
	if _, err := sm.GetPlaybookInspection(); err == nil {
		t.Errorf("expected error from GetPlaybookInspection after unload")
	}

	// Set load error
	sm.SetLoadError(ErrCodePlaybookParseFailed, "parse error details", []string{"error detail 1"})
	if sm.GetStatus() != StatusIdle {
		t.Errorf("expected status %s after load error, got %s", StatusIdle, sm.GetStatus())
	}
	if len(sm.errors) != 1 || sm.errors[0].Code != ErrCodePlaybookParseFailed {
		t.Errorf("expected load error in sm.errors, got %+v", sm.errors)
	}
}
