package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/director"
	"github.com/benedictjohannes/crobe/executor"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func TestServerObserverCallbacks(t *testing.T) {
	sm := NewStateManager("")
	broker := NewEventBroker()
	ch := broker.Subscribe()
	defer broker.Unsubscribe(ch)

	obs := &serverObserver{
		sm:     sm,
		broker: broker,
		runID:  "obs-run-1",
	}

	// 1. OnRunStart
	pb := playbook.Playbook{Title: "Observer Playbook"}
	obs.OnRunStart("obs-run-1", &pb)
	ev := <-ch
	if ev.Type != "state_change" {
		t.Errorf("expected state_change event, got %s", ev.Type)
	}

	// 2. OnSectionStart
	sec := playbook.Section{Title: "Section A"}
	obs.OnSectionStart(sec, 0, 1)
	ev = <-ch
	if ev.Type != "log" {
		t.Errorf("expected log event on section start, got %s", ev.Type)
	}

	// 3. OnAssertionStart
	ass := playbook.Assertion{Code: "A1", Title: "Assertion 1"}
	obs.OnAssertionStart(ass, 0, 1)
	ev = <-ch
	if ev.Type != "assertion_progress" {
		t.Errorf("expected assertion_progress on assertion start, got %s", ev.Type)
	}

	// 4. OnAssertionComplete
	obs.OnAssertionComplete(ass, director.AssertionProgressResult{
		Status:     "passed",
		Passed:     true,
		Score:      1,
		MinScore:   1,
		DurationMs: 10,
	})
	ev = <-ch
	if ev.Type != "assertion_progress" {
		t.Errorf("expected assertion_progress event, got %s", ev.Type)
	}

	// 5. OnLog
	obs.OnLog("Test log line")
	ev = <-ch
	if ev.Type != "log" {
		t.Errorf("expected log event, got %s", ev.Type)
	}

	// 6. OnRunError
	obs.OnRunError(errors.New("something went wrong"))
	ev = <-ch
	if ev.Type != "log" {
		t.Errorf("expected log event on run error, got %s", ev.Type)
	}

	// 7. OnRunCancelled
	obs.OnRunCancelled("obs-run-1", executor.ExecutionTrace{})
	ev = <-ch
	if ev.Type != "execution_cancelled" {
		t.Errorf("expected execution_cancelled event, got %s", ev.Type)
	}

	// 8. OnRunComplete
	obs.OnRunComplete(executor.ExecutionTrace{}, report.FinalReport{})
	ev = <-ch
	if ev.Type != "execution_completed" {
		t.Errorf("expected execution_completed event, got %s", ev.Type)
	}

	if obs.broker.LastEventID() == 0 {
		t.Errorf("expected LastEventID > 0")
	}
}

func TestServerMethodsAndGetters(t *testing.T) {
	// Server with empty token should generate one
	srv, err := NewServer(Config{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	if len(srv.Token()) != 64 {
		t.Errorf("expected generated 64-char token, got %s", srv.Token())
	}
	if srv.StateManager() == nil {
		t.Errorf("expected non-nil StateManager")
	}
	if srv.EventBroker() == nil {
		t.Errorf("expected non-nil EventBroker")
	}
	if srv.LifecycleManager() == nil {
		t.Errorf("expected non-nil LifecycleManager")
	}
	if srv.ShutdownChan() == nil {
		t.Errorf("expected non-nil ShutdownChan")
	}

	// Before Start(), ListeningAddr is default host:port
	if srv.ListeningAddr() != "127.0.0.1:0" {
		t.Errorf("expected 127.0.0.1:0 ListeningAddr before Start(), got %s", srv.ListeningAddr())
	}

	// Start server
	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start() failed: %v", err)
	}

	// Start() twice should return nil (idempotent)
	if err := srv.Start(); err != nil {
		t.Errorf("srv.Start() second time failed: %v", err)
	}

	if srv.ListeningAddr() == "" {
		t.Errorf("expected non-empty ListeningAddr after Start()")
	}
	if srv.URL() == "" {
		t.Errorf("expected non-empty URL after Start()")
	}

	// TriggerShutdown
	srv.TriggerShutdown()
	select {
	case <-srv.ShutdownChan():
		// shutdown channel received signal
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for TriggerShutdown signal")
	}

	// Close() twice should be idempotent
	if err := srv.Close(); err != nil {
		t.Errorf("srv.Close() failed: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Errorf("srv.Close() second time failed: %v", err)
	}
}

func TestHandlers_MethodNotAllowed(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	methodsTests := []struct {
		path    string
		badVerb string
		handler http.HandlerFunc
	}{
		{"/api/state", http.MethodPost, srv.handleState},
		{"/api/playbook/upload", http.MethodGet, srv.handlePlaybookUpload},
		{"/api/playbook/remote", http.MethodGet, srv.handlePlaybookRemote},
		{"/api/playbook", http.MethodPost, srv.handlePlaybookGet},
		{"/api/playbook", http.MethodGet, srv.handlePlaybookDelete},
		{"/api/report/destination", http.MethodGet, srv.handleReportDestinationPut},
		{"/api/run", http.MethodGet, srv.handleRun},
		{"/api/execution/cancel", http.MethodGet, srv.handleCancel},
		{"/api/execution", http.MethodPost, srv.handleExecutionGet},
		{"/api/report", http.MethodPost, srv.handleReportGet},
		{"/api/report/md", http.MethodPost, srv.handleReportMDGet},
		{"/api/report/log", http.MethodPost, srv.handleReportLogGet},
		{"/api/report/download", http.MethodPost, srv.handleReportDownload},
		{"/api/report/remote-submit", http.MethodGet, srv.handleReportRemoteSubmit},
		{"/api/shutdown", http.MethodGet, srv.handleShutdown},
		{"/api/events", http.MethodPost, srv.handleEvents},
	}

	for _, tt := range methodsTests {
		t.Run(tt.path+"_"+tt.badVerb, func(t *testing.T) {
			req := httptest.NewRequest(tt.badVerb, tt.path, nil)
			rec := httptest.NewRecorder()
			tt.handler(rec, req)
			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s with %s returned status %d, want 405", tt.path, tt.badVerb, rec.Code)
			}
		})
	}
}

func TestHandlers_PlaybookUploadEdgeCases(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// 1. Upload when running -> 409 Conflict
	srv.StateManager().SetStatus(StatusRunning)
	req := httptest.NewRequest(http.MethodPost, "/api/playbook/upload", nil)
	rec := httptest.NewRecorder()
	srv.handlePlaybookUpload(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 when running, got %d", rec.Code)
	}

	srv.StateManager().SetStatus(StatusIdle)

	// 2. Upload without multipart boundary -> 400 Bad Request
	req = httptest.NewRequest(http.MethodPost, "/api/playbook/upload", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "multipart/form-data")
	rec = httptest.NewRecorder()
	srv.handlePlaybookUpload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad multipart body, got %d", rec.Code)
	}

	// 3. Upload multipart without 'file' field -> 400 Bad Request
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormField("other_field")
	_, _ = part.Write([]byte("some data"))
	_ = writer.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	srv.handlePlaybookUpload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file field, got %d", rec.Code)
	}

	// 4. Valid JSON playbook upload
	validJSONPlaybook := `{
		"title": "JSON Playbook",
		"sections": [
			{
				"title": "S1",
				"assertions": [
					{
						"code": "J1",
						"title": "J1",
						"cmds": [{"exec": {"script": "echo json"}}]
					}
				]
			}
		]
	}`
	body.Reset()
	writer = multipart.NewWriter(body)
	part, _ = writer.CreateFormFile("file", "playbook.json")
	_, _ = part.Write([]byte(validJSONPlaybook))
	_ = writer.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	srv.handlePlaybookUpload(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid JSON upload, got %d", rec.Code)
	}
	if srv.StateManager().GetStatus() != StatusLoaded {
		t.Errorf("expected status loaded after JSON upload")
	}

	// 5. Invalid JSON playbook upload
	body.Reset()
	writer = multipart.NewWriter(body)
	part, _ = writer.CreateFormFile("file", "bad.json")
	_, _ = part.Write([]byte("{invalid json:"))
	_ = writer.Close()

	req = httptest.NewRequest(http.MethodPost, "/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec = httptest.NewRecorder()
	srv.handlePlaybookUpload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON upload, got %d", rec.Code)
	}
}

func TestHandlers_PlaybookRemoteEdgeCases(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// 1. Conflict when running
	srv.StateManager().SetStatus(StatusRunning)
	req := httptest.NewRequest(http.MethodPost, "/api/playbook/remote", nil)
	rec := httptest.NewRecorder()
	srv.handlePlaybookRemote(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 when running, got %d", rec.Code)
	}

	srv.StateManager().SetStatus(StatusIdle)

	// 2. Invalid JSON request body -> 400
	req = httptest.NewRequest(http.MethodPost, "/api/playbook/remote", bytes.NewReader([]byte("not-json")))
	rec = httptest.NewRecorder()
	srv.handlePlaybookRemote(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json body, got %d", rec.Code)
	}

	// 3. HTTP Server returning 500 error
	remoteErrServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer remoteErrServer.Close()

	RemotePlaybookClient = remoteErrServer.Client()
	defer func() { RemotePlaybookClient = nil }()

	payload, _ := json.Marshal(RemotePlaybookRequest{URL: remoteErrServer.URL + "/playbook.yaml"})
	req = httptest.NewRequest(http.MethodPost, "/api/playbook/remote", bytes.NewReader(payload))
	rec = httptest.NewRecorder()
	srv.handlePlaybookRemote(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for remote server error, got %d", rec.Code)
	}

	// 4. Remote server returning remote JSON playbook
	validJSONPlaybook := `{"title": "Remote JSON", "sections": [{"title": "S1", "assertions": [{"code": "R1", "title": "R1", "cmds": [{"exec": {"script": "echo remote"}}]}]}]}`
	remoteOKServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validJSONPlaybook))
	}))
	defer remoteOKServer.Close()

	RemotePlaybookClient = remoteOKServer.Client()

	payload, _ = json.Marshal(RemotePlaybookRequest{URL: remoteOKServer.URL + "/playbook.json"})
	req = httptest.NewRequest(http.MethodPost, "/api/playbook/remote", bytes.NewReader(payload))
	rec = httptest.NewRecorder()
	srv.handlePlaybookRemote(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for remote JSON fetch, got %d", rec.Code)
	}
	if srv.StateManager().GetPlaybook() == nil || srv.StateManager().GetPlaybook().Title != "Remote JSON" {
		t.Errorf("expected playbook title 'Remote JSON', got %+v", srv.StateManager().GetPlaybook())
	}
}

func TestHandlers_ExecutionEdgeCases(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// 1. Run when no playbook loaded -> 400
	req := httptest.NewRequest(http.MethodPost, "/api/run", nil)
	rec := httptest.NewRecorder()
	srv.handleRun(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for run with no playbook, got %d", rec.Code)
	}

	// 2. Run with validation errors -> 400
	pb := &playbook.Playbook{
		Title: "Val Error Playbook",
		Sections: []playbook.Section{
			{
				Title: "S1",
				Assertions: []playbook.Assertion{
					{Code: "DUP", Title: "D1"},
					{Code: "DUP", Title: "D2"},
				},
			},
		},
	}
	srv.StateManager().SetPlaybook(pb, []byte(""), []playbook.ValidationError{
		{Path: "sections[0].assertions[1].code", Message: "duplicate code DUP"},
	})

	req = httptest.NewRequest(http.MethodPost, "/api/run", nil)
	rec = httptest.NewRecorder()
	srv.handleRun(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for run with validation errors, got %d", rec.Code)
	}

	// 3. Cancel when not running -> 409 Conflict
	srv.StateManager().SetStatus(StatusLoaded)
	req = httptest.NewRequest(http.MethodPost, "/api/execution/cancel", nil)
	rec = httptest.NewRecorder()
	srv.handleCancel(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for cancel when not running, got %d", rec.Code)
	}

	// 4. Cancel when already cancelling -> idempotent 200
	srv.StateManager().SetStatus(StatusRunningCancelling)
	req = httptest.NewRequest(http.MethodPost, "/api/execution/cancel", nil)
	rec = httptest.NewRecorder()
	srv.handleCancel(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent cancel, got %d", rec.Code)
	}
}

func TestHandlers_ReportsEdgeCases(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// 1. Reports when no report exists -> 404
	endpoints := []string{"/api/report", "/api/report/md", "/api/report/log", "/api/report/download"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rec := httptest.NewRecorder()
		if ep == "/api/report" {
			srv.handleReportGet(rec, req)
		} else if ep == "/api/report/md" {
			srv.handleReportMDGet(rec, req)
		} else if ep == "/api/report/log" {
			srv.handleReportLogGet(rec, req)
		} else if ep == "/api/report/download" {
			srv.handleReportDownload(rec, req)
		}
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for %s when no report, got %d", ep, rec.Code)
		}
	}

	// Set report
	dummyRes := report.FinalResult{
		Structured: report.FinalReport{
			OS:         "linux",
			Stats:      report.Stats{Passed: 1, Failed: 0},
			Assertions: make(map[string]report.Assertion),
		},
		Markdown: "# Report Markdown",
		Log:      "=== REPORT LOG ===",
	}
	srv.StateManager().lastReport = &dummyRes

	// 2. Unsupported download format -> 400
	req := httptest.NewRequest(http.MethodGet, "/api/report/download?format=unsupported", nil)
	rec := httptest.NewRecorder()
	srv.handleReportDownload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported download format, got %d", rec.Code)
	}

	// 3. Remote Submit when running -> 409 Conflict
	srv.StateManager().SetStatus(StatusRunning)
	req = httptest.NewRequest(http.MethodPost, "/api/report/remote-submit", nil)
	rec = httptest.NewRecorder()
	srv.handleReportRemoteSubmit(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for remote submit when running, got %d", rec.Code)
	}

	// 4. Remote Submit when cancelled report -> 409 Conflict
	srv.StateManager().SetStatus(StatusCompleted)
	srv.StateManager().isCancelled = true
	req = httptest.NewRequest(http.MethodPost, "/api/report/remote-submit", nil)
	rec = httptest.NewRecorder()
	srv.handleReportRemoteSubmit(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for remote submit when cancelled, got %d", rec.Code)
	}
	srv.StateManager().isCancelled = false

	// 4. Remote Submit when in completed.confirming_submission but no HTTPS configured -> 400
	srv.StateManager().SetStatus(StatusCompletedConfirmingSubmission)
	srv.StateManager().reportDestination.HttpsSource = HttpsSourceOff
	srv.StateManager().reportDestination.HTTPS = nil
	req = httptest.NewRequest(http.MethodPost, "/api/report/remote-submit", nil)
	rec = httptest.NewRecorder()
	srv.handleReportRemoteSubmit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for remote submit without HTTPS config, got %d", rec.Code)
	}
}

func TestHandlers_DestinationPutEdgeCases(t *testing.T) {
	srv, err := NewServer(Config{Token: "testtok1234567890123456789012345678"})
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer srv.Close()

	// 1. Conflict when running
	srv.StateManager().SetStatus(StatusRunning)
	req := httptest.NewRequest(http.MethodPut, "/api/report/destination", nil)
	rec := httptest.NewRecorder()
	srv.handleReportDestinationPut(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 when running, got %d", rec.Code)
	}

	srv.StateManager().SetStatus(StatusLoaded)

	// 2. Bad JSON body -> 400
	req = httptest.NewRequest(http.MethodPut, "/api/report/destination", bytes.NewReader([]byte("not-json")))
	rec = httptest.NewRecorder()
	srv.handleReportDestinationPut(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json body, got %d", rec.Code)
	}
}

func TestLifecycleManagerStopMultipleTimes(t *testing.T) {
	sm := NewStateManager("")
	lm := NewLifecycleManager(sm, LifecycleConfig{
		StartupGracePeriod: 10 * time.Millisecond,
		InactivityTimeout:  10 * time.Millisecond,
	}, func() {})

	lm.Stop()
	lm.Stop() // Should be safe and idempotent
}
