package server

import "github.com/benedictjohannes/crobe/playbook"

// State status constants
const (
	StatusIdle                          = "idle"
	StatusLoaded                        = "loaded"
	StatusRunningElevating              = "running.elevating"
	StatusRunning                       = "running"
	StatusRunningCancelling             = "running.cancelling"
	StatusCompletedConfirmingSubmission = "completed.confirming_submission"
	StatusCompletedSubmitting           = "completed.submitting"
	StatusCompletedSubmitted            = "completed.submitted"
	StatusCompletedSubmissionError      = "completed.submission_error"
	StatusCompleted                     = "completed"
	StatusError                         = "error"
)

// Error codes
const (
	ErrCodePlaybookParseFailed      = "PLAYBOOK_PARSE_FAILED"
	ErrCodePlaybookValidationFailed = "PLAYBOOK_VALIDATION_FAILED"
	ErrCodeRemoteFetchFailed        = "REMOTE_FETCH_FAILED"
	ErrCodeElevationDenied          = "ELEVATION_DENIED"
	ErrCodeElevationTimeout         = "ELEVATION_TIMEOUT"
	ErrCodeElevationFailed          = "ELEVATION_FAILED"
	ErrCodeExecutionAborted         = "EXECUTION_ABORTED"
	ErrCodeExecutionFailed          = "EXECUTION_FAILED"
	ErrCodeFolderWriteFailed        = "FOLDER_WRITE_FAILED"
	ErrCodeRemoteSubmissionFailed   = "REMOTE_SUBMISSION_FAILED"
	ErrCodeRemoteSubmissionTimeout  = "REMOTE_SUBMISSION_TIMEOUT"
	ErrCodeConflict                 = "CONFLICT"
	ErrCodeInternalError            = "INTERNAL_ERROR"
)

// Destination source constants
const (
	FolderSourceDefault  = "default"
	FolderSourceCLI      = "cli"
	FolderSourcePlaybook = "playbook"
	FolderSourceCustom   = "custom"
	FolderSourceOff      = "off"

	HttpsSourcePlaybook = "playbook"
	HttpsSourceCustom   = "custom"
	HttpsSourceOff      = "off"
)

// AppError represents a structured error returned in state and error responses.
type AppError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

// HttpsDestinationConfig represents custom HTTPS submission configuration.
type HttpsDestinationConfig struct {
	URL     string            `json:"url"`
	Format  string            `json:"format,omitempty"` // "json" | "multipart"
	Secret  string            `json:"secret,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// PlaybookHttpsInspection represents sanitized HTTPS configuration from a playbook.
type PlaybookHttpsInspection struct {
	URL                string   `json:"url"`
	Format             string   `json:"format"`
	HasSignatureSecret bool     `json:"hasSignatureSecret"`
	ConfiguredHeaders  []string `json:"configuredHeaders"`
}

// PlaybookDestinationDefaults reflects default destinations defined within the loaded playbook.
type PlaybookDestinationDefaults struct {
	HasFolder  bool                     `json:"has_folder"`
	FolderPath string                   `json:"folder_path,omitempty"`
	HasHTTPS   bool                     `json:"has_https"`
	HTTPS      *PlaybookHttpsInspection `json:"https,omitempty"`
}

// ReportDestinationState represents current active report destination configuration.
type ReportDestinationState struct {
	Folder           string                       `json:"folder,omitempty"`
	FolderSource     string                       `json:"folder_source"`
	HttpsSource      string                       `json:"https_source"`
	HTTPS            *HttpsDestinationConfig      `json:"https,omitempty"`
	PlaybookDefaults *PlaybookDestinationDefaults `json:"playbook_defaults,omitempty"`
}

// AppStateResponse is returned by GET /api/state and state mutation endpoints.
type AppStateResponse struct {
	Status            string                 `json:"status"`
	Errors            []AppError             `json:"errors"`
	ReportDestination ReportDestinationState `json:"report_destination"`
	ActiveRunID       string                 `json:"active_run_id,omitempty"`
}

// PlaybookInspection is returned by GET /api/playbook with full sections, assertions and commands (secrets sanitized).
type PlaybookInspection struct {
	Title                   string                     `json:"title"`
	ReportFrontmatter       map[string]interface{}     `json:"reportFrontmatter,omitempty"`
	Sections                []playbook.Section         `json:"sections"`
	ReportDestination       playbook.ReportDestination `json:"reportDestination,omitempty"`
	ReportDestinationFolder string                     `json:"reportDestinationFolder,omitempty"`
	ReportDestinationHTTPS  *PlaybookHttpsInspection   `json:"reportDestinationHttps,omitempty"`
	RequiresElevation       bool                       `json:"requiresElevation"`
}

// AssertionSnapshot represents live or completed assertion status in an execution snapshot.
type AssertionSnapshot struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	Status     string `json:"status"` // "pending" | "running" | "passed" | "failed" | "cancelled"
	Passed     bool   `json:"passed"`
	Score      int    `json:"score"`
	MinScore   int    `json:"min_score"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
}

// ExecutionSnapshot is returned by GET /api/execution for authoritative state reconstruction.
type ExecutionSnapshot struct {
	RunID               string              `json:"run_id"`
	Status              string              `json:"status"`
	ActiveAssertionCode string              `json:"active_assertion_code,omitempty"`
	LastEventID         int64               `json:"last_event_id"`
	DurationMs          int64               `json:"duration_ms"`
	Assertions          []AssertionSnapshot `json:"assertions"`
	Logs                []string            `json:"logs"`
}

// DestinationUpdateRequest represents payload for PUT /api/report/destination.
type DestinationUpdateRequest struct {
	Folder       *string                 `json:"folder,omitempty"`
	FolderSource *string                 `json:"folder_source,omitempty"`
	HttpsSource  *string                 `json:"https_source,omitempty"`
	HTTPS        *HttpsDestinationConfig `json:"https,omitempty"`
}

// RemotePlaybookRequest represents payload for POST /api/playbook/remote.
type RemotePlaybookRequest struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}
