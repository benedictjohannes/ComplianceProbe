package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func TestStateManager(t *testing.T) {
	sm := NewStateManager("cli-reports-folder", true)

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

func TestStateManager_UI_Destinations(t *testing.T) {
	// 1. Playbook with HTTPS and Folder in UI mode (independent destinations)
	sm := NewStateManager("", true)
	pb := &playbook.Playbook{
		Title:                   "Multi Output Playbook",
		ReportDestination:       playbook.ReportDestinationHTTPS,
		ReportDestinationFolder: "custom-folder",
		ReportDestinationHTTPS: &playbook.ReportDestinationConfig{
			URL: "https://example.com/ingest",
		},
		Sections: []playbook.Section{
			{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
		},
	}
	sm.SetPlaybook(pb, []byte("..."), nil)

	dest := sm.ReportDestination()
	if dest.FolderSource != FolderSourcePlaybook || dest.Folder != resolveFolderPath("custom-folder") {
		t.Errorf("expected FolderSourcePlaybook with resolveFolderPath('custom-folder'), got %s (%s)", dest.FolderSource, dest.Folder)
	}
	if dest.HttpsSource != HttpsSourcePlaybook {
		t.Errorf("expected HttpsSourcePlaybook in UI mode, got %s", dest.HttpsSource)
	}
}

func TestStateManager_DirectCLI_Destinations(t *testing.T) {
	// 1. Playbook specifies HTTPS in Direct CLI mode -> Folder is Off
	sm := NewStateManager("", false)
	pb := &playbook.Playbook{
		Title:             "HTTPS Playbook",
		ReportDestination: playbook.ReportDestinationHTTPS,
		ReportDestinationHTTPS: &playbook.ReportDestinationConfig{
			URL: "https://example.com/ingest",
		},
		Sections: []playbook.Section{
			{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
		},
	}
	sm.SetPlaybook(pb, []byte("..."), nil)

	dest := sm.ReportDestination()
	if dest.FolderSource != FolderSourceOff {
		t.Errorf("expected FolderSourceOff in Direct CLI mode for HTTPS playbook, got %s", dest.FolderSource)
	}
	if dest.HttpsSource != HttpsSourcePlaybook {
		t.Errorf("expected HttpsSourcePlaybook, got %s", dest.HttpsSource)
	}

	// 2. CLI --folder flag overrides HTTPS destination in Direct CLI mode
	smCLI := NewStateManager("/custom/cli/path", false)
	smCLI.SetPlaybook(pb, []byte("..."), nil)

	destCLI := smCLI.ReportDestination()
	if destCLI.FolderSource != FolderSourceCLI || destCLI.Folder != resolveFolderPath("/custom/cli/path") {
		t.Errorf("expected FolderSourceCLI with resolveFolderPath('/custom/cli/path'), got %s (%s)", destCLI.FolderSource, destCLI.Folder)
	}
	if destCLI.HttpsSource != HttpsSourceOff {
		t.Errorf("expected HttpsSourceOff when CLI folder flag overrides, got %s", destCLI.HttpsSource)
	}

	// 3. Playbook default (empty/folder) in Direct CLI mode
	smDefault := NewStateManager("", false)
	pbFolder := &playbook.Playbook{
		Title: "Folder Playbook",
		Sections: []playbook.Section{
			{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
		},
	}
	smDefault.SetPlaybook(pbFolder, []byte("..."), nil)

	destDefault := smDefault.ReportDestination()
	if destDefault.FolderSource != FolderSourceDefault || destDefault.Folder != resolveFolderPath("reports") {
		t.Errorf("expected FolderSourceDefault with resolveFolderPath('reports'), got %s (%s)", destDefault.FolderSource, destDefault.Folder)
	}
	if destDefault.HttpsSource != HttpsSourceOff {
		t.Errorf("expected HttpsSourceOff for default folder playbook, got %s", destDefault.HttpsSource)
	}
}

func TestStateManager_DispatchReport(t *testing.T) {
	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser",
		},
		Markdown: "# Test",
		Log:      "test log",
	}

	t.Run("folder dispatch", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sm-dispatch-folder-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		sm := NewStateManager(tmpDir, false)
		pb := &playbook.Playbook{
			Title: "Test",
			Sections: []playbook.Section{
				{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
			},
		}
		sm.SetPlaybook(pb, []byte("..."), nil)

		if err := sm.DispatchReport(res); err != nil {
			t.Fatalf("DispatchReport failed: %v", err)
		}

		files, _ := os.ReadDir(tmpDir)
		if len(files) != 3 {
			t.Errorf("expected 3 report files, got %d", len(files))
		}
	})

	t.Run("https dispatch success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sm := NewStateManager("", false)
		pb := &playbook.Playbook{
			Title:             "Test HTTPS",
			ReportDestination: playbook.ReportDestinationHTTPS,
			ReportDestinationHTTPS: &playbook.ReportDestinationConfig{
				URL: server.URL,
			},
			Sections: []playbook.Section{
				{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
			},
		}
		sm.SetPlaybook(pb, []byte("..."), nil)

		if err := sm.DispatchReport(res); err != nil {
			t.Fatalf("DispatchReport to HTTPS failed: %v", err)
		}
	})

	t.Run("https dispatch missing config error", func(t *testing.T) {
		sm := NewStateManager("", false)
		pb := &playbook.Playbook{
			Title:             "Test HTTPS",
			ReportDestination: playbook.ReportDestinationHTTPS,
			Sections: []playbook.Section{
				{Title: "S1", Assertions: []playbook.Assertion{{Code: "A1", Title: "A1"}}},
			},
		}
		sm.SetPlaybook(pb, []byte("..."), nil)

		err := sm.DispatchReport(res)
		if err == nil || !strings.Contains(err.Error(), "reportDestinationHttps configuration is missing") {
			t.Errorf("expected error for missing HTTPS config, got %v", err)
		}
	})
}

