package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHttpsDestinationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *HttpsDestinationConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "https configuration is required",
		},
		{
			name: "empty URL",
			cfg: &HttpsDestinationConfig{
				URL: "",
			},
			wantErr: true,
			errMsg:  "https destination URL is required",
		},
		{
			name: "invalid URL scheme (http)",
			cfg: &HttpsDestinationConfig{
				URL: "http://example.com/webhook",
			},
			wantErr: true,
			errMsg:  "HTTPS destination URL must start with https://",
		},
		{
			name: "invalid URL format",
			cfg: &HttpsDestinationConfig{
				URL: "https:// invalid url",
			},
			wantErr: true,
			errMsg:  "invalid https destination URL",
		},
		{
			name: "missing host",
			cfg: &HttpsDestinationConfig{
				URL: "https://",
			},
			wantErr: true,
			errMsg:  "https destination URL must contain a valid host",
		},
		{
			name: "invalid format",
			cfg: &HttpsDestinationConfig{
				URL:    "https://example.com/webhook",
				Format: "xml",
			},
			wantErr: true,
			errMsg:  "https format must be 'json' or 'multipart'",
		},
		{
			name: "invalid header name with spaces",
			cfg: &HttpsDestinationConfig{
				URL: "https://example.com/webhook",
				Headers: map[string]string{
					"Invalid Header": "value",
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP header name",
		},
		{
			name: "invalid header value with CRLF",
			cfg: &HttpsDestinationConfig{
				URL: "https://example.com/webhook",
				Headers: map[string]string{
					"X-Custom-Header": "value\r\nInjected: true",
				},
			},
			wantErr: true,
			errMsg:  "invalid HTTP header value",
		},
		{
			name: "valid minimal config",
			cfg: &HttpsDestinationConfig{
				URL: "https://example.com/webhook",
			},
			wantErr: false,
		},
		{
			name: "valid complete config with json format and headers",
			cfg: &HttpsDestinationConfig{
				URL:    "https://example.com/api/v1/reports",
				Format: "json",
				Secret: "supersecret",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"X-Audit-ID":    "audit-999",
				},
			},
			wantErr: false,
		},
		{
			name: "valid complete config with multipart format",
			cfg: &HttpsDestinationConfig{
				URL:    "https://example.com/api/v1/reports",
				Format: "multipart",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want substring %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestDestinationUpdateRequest_Validate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "crobe-dest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "regular_file.txt")
	if err := os.WriteFile(tempFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		req     DestinationUpdateRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid folder_source enum",
			req: DestinationUpdateRequest{
				FolderSource: strPtr("invalid_source"),
			},
			wantErr: true,
			errMsg:  "invalid folder_source",
		},
		{
			name: "custom folder source with empty folder",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceCustom),
				Folder:       strPtr("   "),
			},
			wantErr: true,
			errMsg:  "custom folder path cannot be empty",
		},
		{
			name: "custom folder path does not exist",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceCustom),
				Folder:       strPtr(filepath.Join(tempDir, "non_existent_subdir")),
			},
			wantErr: true,
			errMsg:  "custom folder path does not exist",
		},
		{
			name: "custom folder path is a file, not directory",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceCustom),
				Folder:       strPtr(tempFile),
			},
			wantErr: true,
			errMsg:  "custom folder path is not a directory",
		},
		{
			name: "valid custom folder path",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceCustom),
				Folder:       strPtr(tempDir),
			},
			wantErr: false,
		},
		{
			name: "valid default folder source",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceDefault),
			},
			wantErr: false,
		},
		{
			name: "valid off folder source",
			req: DestinationUpdateRequest{
				FolderSource: strPtr(FolderSourceOff),
			},
			wantErr: false,
		},
		{
			name: "invalid https_source enum",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr("invalid_https"),
			},
			wantErr: true,
			errMsg:  "invalid https_source",
		},
		{
			name: "custom https source without https config",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr(HttpsSourceCustom),
				HTTPS:       nil,
			},
			wantErr: true,
			errMsg:  "https configuration is required when https_source is 'custom'",
		},
		{
			name: "custom https source with invalid https config",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr(HttpsSourceCustom),
				HTTPS: &HttpsDestinationConfig{
					URL: "http://insecure.com",
				},
			},
			wantErr: true,
			errMsg:  "HTTPS destination URL must start with https://",
		},
		{
			name: "valid custom https source and config",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr(HttpsSourceCustom),
				HTTPS: &HttpsDestinationConfig{
					URL:    "https://example.com/api",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "valid off https source",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr(HttpsSourceOff),
			},
			wantErr: false,
		},
		{
			name: "valid playbook https source",
			req: DestinationUpdateRequest{
				HttpsSource: strPtr(HttpsSourcePlaybook),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want substring %q", err, tt.errMsg)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return filepath.Clean(s) != "" && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || len(s) > 0 && testingContains(s, substr)))
}

func testingContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
