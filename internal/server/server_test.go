package server_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/internal/server"
	"github.com/benedictjohannes/crobe/report"
	"github.com/klauspost/compress/zstd"
)

const sampleValidPlaybookYAML = `
title: "System Integrity Check"
reportFrontmatter:
  environment: "staging"
sections:
  - title: "Disk & Memory"
    description:
      - "Checks disk space and memory"
    assertions:
      - code: "SEC-001"
        title: "Check Free Disk Space"
        description: "Ensures at least some disk exists"
        cmds:
          - exec:
              script: "echo disk ok"
      - code: "SEC-002"
        title: "Check Memory Available"
        description: "Ensures memory command runs"
        cmds:
          - exec:
              script: "echo mem ok"
`

const sampleLongRunningPlaybookYAML = `
title: "Long Running Playbook"
sections:
  - title: "Long Step"
    description:
      - "Takes some time"
    assertions:
      - code: "LONG-001"
        title: "Sleeping Assertion"
        description: "Sleeps to test cancellation"
        cmds:
          - exec:
              script: "sleep 2"
`

const sampleInvalidPlaybookYAML = `
title: "Invalid Playbook"
sections:
  - title: "Duplicate codes"
    description:
      - "Testing duplicate codes"
    assertions:
      - code: "DUP-001"
        title: "Assertion 1"
        description: "First"
        cmds:
          - exec:
              script: "echo 1"
      - code: "DUP-001"
        title: "Assertion 2"
        description: "Second"
        cmds:
          - exec:
              script: "echo 2"
`

func setupTestServer(t *testing.T, cliFolder string) (*server.Server, *httptest.Server) {
	t.Helper()
	cfg := server.Config{
		Host:      "127.0.0.1",
		Port:      0,
		Token:     "test-secret-token-12345678901234567890",
		CLIFolder: cliFolder,
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	testServer := &httptest.Server{
		URL: "http://" + srv.ListeningAddr(),
	}

	t.Cleanup(func() {
		_ = srv.Close()
	})

	return srv, testServer
}

func TestMain(m *testing.M) {
	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	os.Exit(m.Run())
}

func authedRequest(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

func TestServer_AuthAndSecurity(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	// 1. Root without token query or auth header serves index.html
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / returned status %d, want 200", resp.StatusCode)
	}
	if csp := resp.Header.Get("Content-Security-Policy"); csp == "" {
		t.Errorf("missing Content-Security-Policy header")
	} else if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Errorf("Content-Security-Policy header %q missing required directives", csp)
	}
	if cache := resp.Header.Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, want 'no-store'", cache)
	}

	// 2. Token Bootstrap redirect: GET /?token=<valid>
	clientNoRedirect := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/?token="+srv.Token(), nil)
	resp, err = clientNoRedirect.Do(req)
	if err != nil {
		t.Fatalf("GET /?token=... failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("GET /?token=... status = %d, want 303 See Other", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want '/'", loc)
	}
	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == server.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value != srv.Token() {
		t.Errorf("expected session cookie with token value, got %+v", sessionCookie)
	}

	// 3. Token Bootstrap redirect: GET /?token=<invalid>
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/?token=wrong-token", nil)
	resp, err = clientNoRedirect.Do(req)
	if err != nil {
		t.Fatalf("GET /?token=wrong failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /?token=wrong status = %d, want 401", resp.StatusCode)
	}

	// 4. API access without Auth -> 401
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/state failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated GET /api/state status = %d, want 401", resp.StatusCode)
	}

	// 5. API access with Bearer header -> 200
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("authenticated GET /api/state failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("authenticated GET /api/state status = %d, want 200", resp.StatusCode)
	}

	// 6. API access with Cookie -> 200
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
	req.AddCookie(sessionCookie)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cookie authenticated GET /api/state failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("cookie authenticated GET /api/state status = %d, want 200", resp.StatusCode)
	}
}

func TestServer_PlaybookUploadAndInspection(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	// 1. Initial State is idle
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get state: %v", err)
	}
	var stateResp server.AppStateResponse
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusIdle {
		t.Errorf("initial status = %s, want %s", stateResp.Status, server.StatusIdle)
	}

	// GET /api/playbook on idle -> 404
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/playbook", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /api/playbook on idle returned %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Upload valid YAML playbook
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "playbook.yaml")
	_, _ = part.Write([]byte(sampleValidPlaybookYAML))
	_ = writer.Close()

	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d, want 200", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusLoaded {
		t.Errorf("status after upload = %s, want %s", stateResp.Status, server.StatusLoaded)
	}
	if len(stateResp.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(stateResp.Errors))
	}

	// 3. GET /api/playbook returns inspection data
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/playbook", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/playbook failed: %v", err)
	}
	var inspection server.PlaybookInspection
	_ = json.NewDecoder(resp.Body).Decode(&inspection)
	resp.Body.Close()

	if inspection.Title != "System Integrity Check" {
		t.Errorf("inspection title = %q, want 'System Integrity Check'", inspection.Title)
	}
	if len(inspection.Sections) != 1 || len(inspection.Sections[0].Assertions) != 2 {
		t.Errorf("unexpected sections/assertions: %+v", inspection.Sections)
	} else {
		firstAss := inspection.Sections[0].Assertions[0]
		if len(firstAss.Cmds) != 1 || firstAss.Cmds[0].Exec.Script != "echo disk ok" {
			t.Errorf("expected cmds[0].exec.script to be 'echo disk ok', got %+v", firstAss.Cmds)
		}
	}

	// 4. Upload invalid YAML syntax -> stays idle, errors populated
	body.Reset()
	writer = multipart.NewWriter(body)
	part, _ = writer.CreateFormFile("file", "bad.yaml")
	_, _ = part.Write([]byte("title: [unclosed list"))
	_ = writer.Close()

	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad upload status = %d, want 400", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusIdle {
		t.Errorf("status after bad upload = %s, want %s", stateResp.Status, server.StatusIdle)
	}
	if len(stateResp.Errors) == 0 || stateResp.Errors[0].Code != server.ErrCodePlaybookParseFailed {
		t.Errorf("expected PLAYBOOK_PARSE_FAILED error, got: %+v", stateResp.Errors)
	}

	// 5. Upload YAML with validation failure -> status loaded, but errors populated
	body.Reset()
	writer = multipart.NewWriter(body)
	part, _ = writer.CreateFormFile("file", "duplicate.yaml")
	_, _ = part.Write([]byte(sampleInvalidPlaybookYAML))
	_ = writer.Close()

	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusLoaded {
		t.Errorf("status after validation fail upload = %s, want loaded", stateResp.Status)
	}
	if len(stateResp.Errors) == 0 || stateResp.Errors[0].Code != server.ErrCodePlaybookValidationFailed {
		t.Errorf("expected PLAYBOOK_VALIDATION_FAILED error, got: %+v", stateResp.Errors)
	}

	// 6. Delete Playbook -> resets to idle
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/playbook", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("DELETE /api/playbook returned %d, want 200", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusIdle {
		t.Errorf("status after DELETE = %s, want idle", stateResp.Status)
	}
}

func TestServer_PlaybookRemoteFetch(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	// Set up mock TLS remote server
	remoteTLS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-remote-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte(sampleValidPlaybookYAML))
	}))
	defer remoteTLS.Close()

	server.RemotePlaybookClient = remoteTLS.Client()
	defer func() { server.RemotePlaybookClient = nil }()

	// 1. Insecure HTTP fetch is blocked
	reqPayload, _ := json.Marshal(server.RemotePlaybookRequest{
		URL: "http://insecure.com/playbook.yaml",
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/remote", bytes.NewReader(reqPayload))
	authedRequest(req, srv.Token())
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("insecure remote fetch status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Valid HTTPS fetch
	reqPayload, _ = json.Marshal(server.RemotePlaybookRequest{
		URL: remoteTLS.URL + "/playbook.yaml",
		Headers: map[string]string{
			"Authorization": "Bearer secret-remote-token",
		},
	})
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/remote", bytes.NewReader(reqPayload))
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("remote fetch request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("remote fetch status = %d, want 200", resp.StatusCode)
	}
	var stateResp server.AppStateResponse
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusLoaded {
		t.Errorf("status after remote fetch = %s, want loaded", stateResp.Status)
	}
}

func TestServer_DestinationUpdates(t *testing.T) {
	// Server without CLI folder flag
	srv, ts := setupTestServer(t, "")

	// Load a playbook
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "playbook.yaml")
	_, _ = part.Write([]byte(sampleValidPlaybookYAML))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	// 1. Update Destination successfully
	customFolder := "custom_reports_dir"
	folderSource := server.FolderSourceCustom
	httpsSource := server.HttpsSourceCustom
	updatePayload, _ := json.Marshal(server.DestinationUpdateRequest{
		Folder:       &customFolder,
		FolderSource: &folderSource,
		HttpsSource:  &httpsSource,
		HTTPS: &server.HttpsDestinationConfig{
			URL:    "https://example.com/webhook",
			Format: "json",
		},
	})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/report/destination", bytes.NewReader(updatePayload))
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("destination update failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("destination update status = %d, want 200", resp.StatusCode)
	}
	var stateResp server.AppStateResponse
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	expectedFolder, _ := filepath.Abs(customFolder)
	if stateResp.ReportDestination.Folder != expectedFolder {
		t.Errorf("folder = %q, want %q", stateResp.ReportDestination.Folder, expectedFolder)
	}
	if stateResp.ReportDestination.HttpsSource != server.HttpsSourceCustom {
		t.Errorf("https_source = %s, want %s", stateResp.ReportDestination.HttpsSource, server.HttpsSourceCustom)
	}

	// 2. Invalid HTTPS URL (insecure scheme) -> 400
	badHTTPS := &server.HttpsDestinationConfig{
		URL: "http://insecure.com/webhook",
	}
	updatePayload, _ = json.Marshal(server.DestinationUpdateRequest{
		HTTPS: badHTTPS,
	})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/report/destination", bytes.NewReader(updatePayload))
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("insecure https destination update status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Test CLI Locked Folder Flag
	tmpFolder, _ := os.MkdirTemp("", "crobe-test-cli-*")
	defer os.RemoveAll(tmpFolder)
	srvCLI, tsCLI := setupTestServer(t, tmpFolder)

	// Attempting to modify folder when locked by CLI flag -> 400
	newFolder := "some_other_folder"
	updatePayload, _ = json.Marshal(server.DestinationUpdateRequest{
		Folder: &newFolder,
	})
	req, _ = http.NewRequest(http.MethodPut, tsCLI.URL+"/api/report/destination", bytes.NewReader(updatePayload))
	authedRequest(req, srvCLI.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("modifying CLI-locked folder status = %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_ExecutionRunAndReports(t *testing.T) {
	tempReportsDir, err := os.MkdirTemp("", "crobe-reports-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempReportsDir)

	srv, ts := setupTestServer(t, tempReportsDir)

	// 1. Upload valid playbook
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "playbook.yaml")
	_, _ = part.Write([]byte(sampleValidPlaybookYAML))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	// 2. Trigger Run: POST /api/run
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/run", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/run failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /api/run status = %d, want 202 Accepted", resp.StatusCode)
	}
	var runResp map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&runResp)
	resp.Body.Close()
	runID := runResp["run_id"]
	if runID == "" {
		t.Errorf("expected run_id in response")
	}

	// 3. Wait for execution to finish
	var finalState server.AppStateResponse
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
		authedRequest(req, srv.Token())
		resp, _ = http.DefaultClient.Do(req)
		_ = json.NewDecoder(resp.Body).Decode(&finalState)
		resp.Body.Close()

		if finalState.Status == server.StatusCompleted || finalState.Status == server.StatusCompletedConfirmingSubmission {
			break
		}
	}

	if finalState.Status != server.StatusCompleted {
		t.Fatalf("final status = %s, want %s", finalState.Status, server.StatusCompleted)
	}

	// 4. Test Authoritative Snapshot: GET /api/execution
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/execution", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/execution failed: %v", err)
	}
	var snapshot server.ExecutionSnapshot
	_ = json.NewDecoder(resp.Body).Decode(&snapshot)
	resp.Body.Close()

	if snapshot.RunID != runID {
		t.Errorf("snapshot RunID = %s, want %s", snapshot.RunID, runID)
	}
	if len(snapshot.Assertions) != 2 {
		t.Errorf("expected 2 assertions in snapshot, got %d", len(snapshot.Assertions))
	}
	for _, a := range snapshot.Assertions {
		if a.Status != "passed" || !a.Passed {
			t.Errorf("expected assertion %s to pass, got %+v", a.Code, a)
		}
	}

	// 5. Test Report Endpoints
	// JSON report
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/report failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/report status = %d, want 200", resp.StatusCode)
	}
	var finalRep report.FinalReport
	_ = json.NewDecoder(resp.Body).Decode(&finalRep)
	resp.Body.Close()
	if finalRep.Stats.Passed != 2 {
		t.Errorf("stats.passed = %d, want 2", finalRep.Stats.Passed)
	}

	// Markdown report
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/md?download=1", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/report/md status = %d, want 200", resp.StatusCode)
	}
	if disp := resp.Header.Get("Content-Disposition"); !strings.Contains(disp, "attachment") {
		t.Errorf("expected Content-Disposition header with attachment, got %q", disp)
	}
	mdBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(mdBytes), "System Integrity Check") {
		t.Errorf("markdown report missing title")
	}

	// Log report
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/log", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/report/log status = %d, want 200", resp.StatusCode)
	}
	logBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(logBytes), "REPORT LOG") {
		t.Errorf("log report missing REPORT LOG header")
	}

	// 6. Test Download Bundle: zip, tar, tar.gz
	// Zip
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/download?format=zip", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("download zip status = %d, want 200", resp.StatusCode)
	}
	zipData, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("invalid zip archive: %v", err)
	}
	if len(zr.File) != 3 {
		t.Errorf("expected 3 files in zip, got %d", len(zr.File))
	}

	// Tar.gz
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/download?format=tar.gz", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("download tar.gz status = %d, want 200", resp.StatusCode)
	}
	tgzData, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	gr, err := gzip.NewReader(bytes.NewReader(tgzData))
	if err != nil {
		t.Fatalf("invalid gzip reader: %v", err)
	}
	tr := tar.NewReader(gr)
	fileCount := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		fileCount++
	}
	if fileCount != 3 {
		t.Errorf("expected 3 files in tar.gz, got %d", fileCount)
	}

	// Tar.zst
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/download?format=tar.zst", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("download tar.zst status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/zstd" {
		t.Errorf("download tar.zst content-type = %s, want application/zstd", ct)
	}
	tzstData, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	zrZstd, err := zstd.NewReader(bytes.NewReader(tzstData))
	if err != nil {
		t.Fatalf("invalid zstd reader: %v", err)
	}
	tr = tar.NewReader(zrZstd)
	fileCount = 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		fileCount++
	}
	zrZstd.Close()
	if fileCount != 3 {
		t.Errorf("expected 3 files in tar.zst, got %d", fileCount)
	}
}

func TestServer_ExecutionCancellation(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	// 1. Upload long-running playbook
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "long.yaml")
	_, _ = part.Write([]byte(sampleLongRunningPlaybookYAML))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	// 2. Start Run
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/run", nil)
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/run failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /api/run status = %d, want 202", resp.StatusCode)
	}
	resp.Body.Close()

	// Wait briefly so execution starts
	time.Sleep(100 * time.Millisecond)

	// 3. Cancel Execution
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/execution/cancel", nil)
	authedRequest(req, srv.Token())
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/execution/cancel failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("cancel status = %d, want 200", resp.StatusCode)
	}
	var stateResp server.AppStateResponse
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusLoaded {
		t.Errorf("status after cancellation unwound = %s, want loaded", stateResp.Status)
	}
	hasAbortedError := false
	for _, e := range stateResp.Errors {
		if e.Code == server.ErrCodeExecutionAborted {
			hasAbortedError = true
			break
		}
	}
	if !hasAbortedError {
		t.Errorf("expected EXECUTION_ABORTED error in state, got: %+v", stateResp.Errors)
	}

	// 4. On cancelled run: GET /api/report and GET /api/report/download return 409 Conflict
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("GET /api/report on cancelled run returned %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/download", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("GET /api/report/download on cancelled run returned %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()

	// 5. Partial markdown and logs remain accessible
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/report/md", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/report/md on cancelled run returned %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestServer_RemoteReportSubmissionAndRetry(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	failOnce := true
	remoteSubmissionServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		if failOnce {
			failOnce = false
			http.Error(w, "Temporary Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"received"}`))
	}))
	defer remoteSubmissionServer.Close()

	cli := remoteSubmissionServer.Client()
	cli.Timeout = 5 * time.Second
	reportwriter.HTTPClient = cli
	defer func() { reportwriter.HTTPClient = nil }()

	// 1. Upload playbook
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "playbook.yaml")
	_, _ = part.Write([]byte(sampleValidPlaybookYAML))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/playbook/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	authedRequest(req, srv.Token())
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	// 2. Set Custom HTTPS Destination
	folderOff := server.FolderSourceOff
	httpsCustom := server.HttpsSourceCustom
	updatePayload, _ := json.Marshal(server.DestinationUpdateRequest{
		FolderSource: &folderOff,
		HttpsSource:  &httpsCustom,
		HTTPS: &server.HttpsDestinationConfig{
			URL:    remoteSubmissionServer.URL + "/submit",
			Format: "json",
		},
	})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/report/destination", bytes.NewReader(updatePayload))
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	// 3. Run execution
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/run", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	// Wait for execution to transition to completed.confirming_submission
	var stateResp server.AppStateResponse
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/state", nil)
		authedRequest(req, srv.Token())
		resp, _ = http.DefaultClient.Do(req)
		_ = json.NewDecoder(resp.Body).Decode(&stateResp)
		resp.Body.Close()

		if stateResp.Status == server.StatusCompletedConfirmingSubmission {
			break
		}
	}

	if stateResp.Status != server.StatusCompletedConfirmingSubmission {
		t.Fatalf("expected status completed.confirming_submission, got %s", stateResp.Status)
	}

	// 4. Remote Submit fails on first attempt -> 502 Bad Gateway, returns to completed.submission_error
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/report/remote-submit", nil)
	authedRequest(req, srv.Token())
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("failed remote submit status = %d, want 502", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusCompletedSubmissionError {
		t.Errorf("status after failed submit = %s, want completed.submission_error", stateResp.Status)
	}

	// 5. Retry Remote Submit -> succeeds, transitions to completed.submitted
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/report/remote-submit", nil)
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("retry remote submit failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("retry remote submit status = %d, want 200", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()

	if stateResp.Status != server.StatusCompletedSubmitted {
		t.Errorf("status after successful submit = %s, want completed.submitted", stateResp.Status)
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	srv, ts := setupTestServer(t, "")

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/shutdown", nil)
	authedRequest(req, srv.Token())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/shutdown failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /api/shutdown status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	select {
	case <-srv.ShutdownChan():
		// Shutdown signal received
	case <-time.After(3 * time.Second):
		t.Errorf("timed out waiting for shutdown signal")
	}
}
