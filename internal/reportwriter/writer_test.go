package reportwriter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func TestDispatchReport(t *testing.T) {
	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser",
		},
		Markdown: "# Test",
		Log:      "test log",
	}

	t.Run("folder destination", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dispatch-folder-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		oldDir := DefaultReportsDir
		DefaultReportsDir = tmpDir
		defer func() { DefaultReportsDir = oldDir }()

		config := &playbook.Playbook{
			ReportDestination: playbook.ReportDestinationFolder,
		}
		err = DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport to folder failed: %v", err)
		}

		// Verify files exist in tmpDir
		files, _ := os.ReadDir(tmpDir)
		if len(files) != 3 {
			t.Errorf("Expected 3 files in reports directory, got %d", len(files))
		}
	})

	t.Run("folder destination using config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dispatch-config-folder-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Ensure DefaultReportsDir is empty to trigger the branch:
		// if reportsDir == "" { reportsDir = config.ReportDestinationFolder }
		oldDir := DefaultReportsDir
		DefaultReportsDir = ""
		defer func() { DefaultReportsDir = oldDir }()

		config := &playbook.Playbook{
			ReportDestination:       playbook.ReportDestinationFolder,
			ReportDestinationFolder: tmpDir,
		}
		err = DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport to config folder failed: %v", err)
		}

		// Verify files exist in tmpDir
		files, _ := os.ReadDir(tmpDir)
		if len(files) != 3 {
			t.Errorf("Expected 3 files in reports directory, got %d", len(files))
		}
	})

	t.Run("default destination (empty)", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dispatch-empty-dest-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		oldDir := DefaultReportsDir
		DefaultReportsDir = tmpDir
		defer func() { DefaultReportsDir = oldDir }()

		config := &playbook.Playbook{
			ReportDestination: "", // Should default to folder
		}
		err = DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport with empty destination failed: %v", err)
		}

		files, _ := os.ReadDir(tmpDir)
		if len(files) != 3 {
			t.Errorf("Expected 3 files, got %d", len(files))
		}
	})

	t.Run("unknown destination", func(t *testing.T) {
		config := &playbook.Playbook{
			ReportDestination: "somewhere-else",
		}
		err := DispatchReport(config, res)
		if err == nil || !strings.Contains(err.Error(), "unknown reportDestination") {
			t.Errorf("Expected error for unknown destination, got %v", err)
		}
	})

	t.Run("https missing URL", func(t *testing.T) {
		config := &playbook.Playbook{
			ReportDestination: playbook.ReportDestinationHTTPS,
		}
		err := DispatchReport(config, res)
		if err == nil || !strings.Contains(err.Error(), "reportDestinationHttps is missing") {
			t.Errorf("Expected error for missing URL, got %v", err)
		}
	})

	t.Run("https success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &playbook.Playbook{
			ReportDestination: playbook.ReportDestinationHTTPS,
			ReportDestinationHTTPS: &playbook.ReportDestinationConfig{
				URL: server.URL,
			},
		}
		err := DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport to HTTPS failed: %v", err)
		}
	})
}

func TestWriteToFolder(t *testing.T) {
	t.Run("MkdirAll failure", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "mkdir-fail-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a directory that is searchable (x) but not writable (w)
		unwritableDir := filepath.Join(tmpDir, "unwritable")
		if err := os.Mkdir(unwritableDir, 0555); err != nil {
			t.Fatal(err)
		}

		// Trying to create a subdirectory inside an unwritable parent will fail in MkdirAll
		err = WriteToFolder(filepath.Join(unwritableDir, "reports"), report.FinalResult{})
		if err == nil || !strings.Contains(err.Error(), "failed to create reports directory") {
			t.Errorf("Expected MkdirAll failure, got %v", err)
		}
	})

	tmpDir, err := os.MkdirTemp("", "reportwriter-write-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser",
		},
		Markdown: "# Test Markdown",
		Log:      "test log",
	}

	err = WriteToFolder(tmpDir, res)
	if err != nil {
		t.Fatalf("WriteToFolder failed: %v", err)
	}

	// Verify files exist
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files in reports directory, got %d", len(files))
	}

	foundLog := false
	foundMD := false
	foundJSON := false

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".log") {
			foundLog = true
		}
		if strings.HasSuffix(f.Name(), ".md") {
			foundMD = true
		}
		if strings.HasSuffix(f.Name(), ".json") {
			foundJSON = true
		}
	}

	if !foundLog || !foundMD || !foundJSON {
		t.Errorf("Missing report files: log=%v, md=%v, json=%v", foundLog, foundMD, foundJSON)
	}

	t.Run("default reports directory", func(t *testing.T) {
		// This will create a "reports" folder in the current directory.
		err := WriteToFolder("", res)
		if err != nil {
			t.Fatalf("WriteToFolder with default dir failed: %v", err)
		}
		defer os.RemoveAll("reports")

		if _, err := os.Stat("reports"); os.IsNotExist(err) {
			t.Errorf("Expected 'reports' directory to be created, but it's missing")
		}
	})
}

func TestWriteToFolder_Errors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reportwriter-error-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file where the directory should be
	errPath := filepath.Join(tmpDir, "somefile")
	os.WriteFile(errPath, []byte("not a directory"), 0644)

	res := report.FinalResult{}
	err = WriteToFolder(errPath, res)
	if err == nil {
		t.Errorf("Expected error when target directory is a file, got nil")
	}
}

func TestWriteToHTTP_Errors(t *testing.T) {
	res := report.FinalResult{}

	t.Run("insecure URL", func(t *testing.T) {
		config := &playbook.ReportDestinationConfig{
			URL: "http://example.com",
		}
		err := WriteToHTTP(config, res)
		if err == nil || !strings.Contains(err.Error(), "insecure HTTP report submission is not allowed") {
			t.Errorf("Expected error for insecure URL, got %v", err)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		config := &playbook.ReportDestinationConfig{
			URL: "https://invalid space",
		}
		err := WriteToHTTP(config, res)
		if err == nil {
			t.Errorf("Expected error for invalid URL, got nil")
		}
	})

	t.Run("server error status", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		config := &playbook.ReportDestinationConfig{
			URL: server.URL,
		}
		err := WriteToHTTP(config, res)
		if err == nil || !strings.Contains(err.Error(), "status 500") {
			t.Errorf("Expected error for 500 status, got %v", err)
		}
	})

	t.Run("connection error", func(t *testing.T) {
		// Create a server and immediately close it
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		config := &playbook.ReportDestinationConfig{
			URL: url,
		}
		err := WriteToHTTP(config, res)
		if err == nil {
			t.Errorf("Expected connection error, got nil")
		}
	})
}
