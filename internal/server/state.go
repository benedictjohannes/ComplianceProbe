package server

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
)

func resolveFolderPath(dir string) string {
	if dir == "" {
		dir = "reports"
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// StateManager is the central thread-safe state container.
type StateManager struct {
	mu sync.Mutex

	status            string
	errors            []AppError
	reportDestination ReportDestinationState
	cliFolder         string
	isUI              bool

	playbook    *playbook.Playbook
	playbookRaw []byte

	activeRunID     string
	activeRunCancel context.CancelFunc
	activeRunDone   chan struct{}
	isCancelled     bool
	runStartTime    time.Time

	snapshot   ExecutionSnapshot
	lastReport *report.FinalResult
}

// NewStateManager creates a new initialized StateManager.
func NewStateManager(cliFolder string, isUI bool) *StateManager {
	folderSource := FolderSourceDefault
	folderPath := resolveFolderPath(cliFolder)
	if cliFolder != "" {
		folderSource = FolderSourceCLI
	}

	return &StateManager{
		status: StatusIdle,
		errors: make([]AppError, 0),
		reportDestination: ReportDestinationState{
			Folder:       folderPath,
			FolderSource: folderSource,
			HttpsSource:  HttpsSourceOff,
		},
		cliFolder: cliFolder,
		isUI:      isUI,
		snapshot: ExecutionSnapshot{
			Status:     StatusIdle,
			Assertions: make([]AssertionSnapshot, 0),
			Logs:       make([]string, 0),
		},
	}
}

// getStateResponseLocked returns the state summary assuming sm.mu is already held.
func (sm *StateManager) getStateResponseLocked() AppStateResponse {
	errorsCopy := make([]AppError, len(sm.errors))
	copy(errorsCopy, sm.errors)

	return AppStateResponse{
		Status:            sm.status,
		Errors:            errorsCopy,
		ReportDestination: sm.reportDestination,
		ActiveRunID:       sm.activeRunID,
	}
}

// GetStateResponse returns the current state summary.
func (sm *StateManager) GetStateResponse() AppStateResponse {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.getStateResponseLocked()
}

// ReportDestination returns the active report destination configuration.
func (sm *StateManager) ReportDestination() ReportDestinationState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.reportDestination
}

// CanMutate checks if mutation requests are allowed.
func (sm *StateManager) CanMutate() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.status != StatusRunningElevating &&
		sm.status != StatusRunning &&
		sm.status != StatusRunningCancelling &&
		sm.status != StatusCompletedSubmitting
}

// SetPlaybook updates the loaded playbook, initializes inspection defaults, and updates validation status.
func (sm *StateManager) SetPlaybook(pb *playbook.Playbook, raw []byte, valErrors playbook.ValidationErrors) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.playbook = pb
	sm.playbookRaw = raw
	sm.status = StatusLoaded
	sm.errors = make([]AppError, 0)
	sm.isCancelled = false
	sm.lastReport = nil
	sm.activeRunID = ""

	// Build PlaybookDestinationDefaults
	var httpsInspection *PlaybookHttpsInspection
	hasHttps := pb.ReportDestinationHTTPS != nil
	if hasHttps {
		headers := make([]string, 0, len(pb.ReportDestinationHTTPS.AdditionalHeaders))
		for k := range pb.ReportDestinationHTTPS.AdditionalHeaders {
			headers = append(headers, k)
		}
		format := string(pb.ReportDestinationHTTPS.Format)
		if format == "" {
			format = "multipart"
		}
		httpsInspection = &PlaybookHttpsInspection{
			URL:                pb.ReportDestinationHTTPS.URL,
			Format:             format,
			HasSignatureSecret: pb.ReportDestinationHTTPS.SignatureSecret != "",
			ConfiguredHeaders:  headers,
		}
	}

	defaults := &PlaybookDestinationDefaults{
		HasFolder:  pb.ReportDestinationFolder != "",
		FolderPath: resolveFolderPath(pb.ReportDestinationFolder),
		HasHTTPS:   hasHttps,
		HTTPS:      httpsInspection,
	}
	sm.reportDestination.PlaybookDefaults = defaults

	if sm.isUI {
		// Web UI mode: Independent destinations
		// Set folder destination
		if sm.cliFolder != "" {
			sm.reportDestination.FolderSource = FolderSourceCLI
			sm.reportDestination.Folder = resolveFolderPath(sm.cliFolder)
		} else if pb.ReportDestinationFolder != "" {
			sm.reportDestination.FolderSource = FolderSourcePlaybook
			sm.reportDestination.Folder = resolveFolderPath(pb.ReportDestinationFolder)
		} else {
			sm.reportDestination.FolderSource = FolderSourceDefault
			sm.reportDestination.Folder = resolveFolderPath("reports")
		}

		// Set HTTPS destination
		if pb.ReportDestination == playbook.ReportDestinationHTTPS {
			sm.reportDestination.HttpsSource = HttpsSourcePlaybook
		} else {
			sm.reportDestination.HttpsSource = HttpsSourceOff
		}
	} else {
		// Direct CLI mode: Mutually exclusive destinations
		if sm.cliFolder != "" {
			// CLI flag --folder explicitly overrides destination to folder
			sm.reportDestination.FolderSource = FolderSourceCLI
			sm.reportDestination.Folder = resolveFolderPath(sm.cliFolder)
			sm.reportDestination.HttpsSource = HttpsSourceOff
		} else if pb.ReportDestination == playbook.ReportDestinationHTTPS {
			// Playbook specifies HTTPS
			sm.reportDestination.FolderSource = FolderSourceOff
			sm.reportDestination.HttpsSource = HttpsSourcePlaybook
		} else {
			// Folder destination (playbook or default)
			sm.reportDestination.HttpsSource = HttpsSourceOff
			if pb.ReportDestinationFolder != "" {
				sm.reportDestination.FolderSource = FolderSourcePlaybook
				sm.reportDestination.Folder = resolveFolderPath(pb.ReportDestinationFolder)
			} else {
				sm.reportDestination.FolderSource = FolderSourceDefault
				sm.reportDestination.Folder = resolveFolderPath("reports")
			}
		}
	}

	// Initialize snapshot assertions
	var assertions []AssertionSnapshot
	for _, sec := range pb.Sections {
		for _, ass := range sec.Assertions {
			assertions = append(assertions, AssertionSnapshot{
				Code:     ass.Code,
				Title:    ass.Title,
				Status:   "pending",
				MinScore: ass.GetMinPassingScore(),
			})
		}
	}
	sm.snapshot = ExecutionSnapshot{
		Status:      StatusLoaded,
		LastEventID: 0,
		Assertions:  assertions,
		Logs:        make([]string, 0),
	}

	// Handle validation errors if any
	if len(valErrors) > 0 {
		sm.errors = append(sm.errors, AppError{
			Code:    ErrCodePlaybookValidationFailed,
			Message: "Playbook validation failed",
			Detail:  valErrors,
		})
	}
}

// SetLoadError handles parse or remote fetch failure, keeping state idle.
func (sm *StateManager) SetLoadError(code, message string, detail interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.status = StatusIdle
	sm.playbook = nil
	sm.playbookRaw = nil
	sm.errors = []AppError{
		{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}
	sm.snapshot = ExecutionSnapshot{
		Status:     StatusIdle,
		Assertions: make([]AssertionSnapshot, 0),
		Logs:       make([]string, 0),
	}
}

// UnloadPlaybook resets state to idle.
func (sm *StateManager) UnloadPlaybook() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.status == StatusRunningElevating ||
		sm.status == StatusRunning ||
		sm.status == StatusRunningCancelling ||
		sm.status == StatusCompletedSubmitting {
		return fmt.Errorf("cannot unload playbook during execution or submission")
	}

	sm.status = StatusIdle
	sm.playbook = nil
	sm.playbookRaw = nil
	sm.errors = make([]AppError, 0)
	sm.lastReport = nil
	sm.isCancelled = false
	sm.activeRunID = ""

	folderSource := FolderSourceDefault
	folderPath := resolveFolderPath(sm.cliFolder)
	if sm.cliFolder != "" {
		folderSource = FolderSourceCLI
	}

	sm.reportDestination = ReportDestinationState{
		Folder:       folderPath,
		FolderSource: folderSource,
		HttpsSource:  HttpsSourceOff,
	}

	sm.snapshot = ExecutionSnapshot{
		Status:     StatusIdle,
		Assertions: make([]AssertionSnapshot, 0),
		Logs:       make([]string, 0),
	}
	return nil
}

// GetPlaybookInspection returns the playbook structure with sanitized secrets.
func (sm *StateManager) GetPlaybookInspection() (*PlaybookInspection, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.playbook == nil {
		return nil, fmt.Errorf("no playbook loaded")
	}

	pb := sm.playbook
	var httpsInspection *PlaybookHttpsInspection
	if pb.ReportDestinationHTTPS != nil {
		headers := make([]string, 0, len(pb.ReportDestinationHTTPS.AdditionalHeaders))
		for k := range pb.ReportDestinationHTTPS.AdditionalHeaders {
			headers = append(headers, k)
		}
		format := string(pb.ReportDestinationHTTPS.Format)
		if format == "" {
			format = "multipart"
		}
		httpsInspection = &PlaybookHttpsInspection{
			URL:                pb.ReportDestinationHTTPS.URL,
			Format:             format,
			HasSignatureSecret: pb.ReportDestinationHTTPS.SignatureSecret != "",
			ConfiguredHeaders:  headers,
		}
	}

	return &PlaybookInspection{
		Title:                   pb.Title,
		ReportFrontmatter:       pb.ReportFrontmatter,
		Sections:                pb.Sections,
		ReportDestination:       pb.ReportDestination,
		ReportDestinationFolder: pb.ReportDestinationFolder,
		ReportDestinationHTTPS:  httpsInspection,
		RequiresElevation:       pb.RequiresElevation(),
	}, nil
}

// UpdateDestination updates the report destination settings.
func (sm *StateManager) UpdateDestination(req DestinationUpdateRequest) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.status == StatusRunningElevating ||
		sm.status == StatusRunning ||
		sm.status == StatusRunningCancelling ||
		sm.status == StatusCompletedSubmitting {
		return fmt.Errorf("cannot update destination during execution or submission")
	}

	// If CLI flag locked folder, prevent modifying folder
	resolvedCLI := resolveFolderPath(sm.cliFolder)
	if sm.cliFolder != "" && (req.FolderSource != nil && *req.FolderSource != FolderSourceCLI || req.Folder != nil && resolveFolderPath(*req.Folder) != resolvedCLI) {
		return fmt.Errorf("cannot modify destination: folder is locked by --folder CLI flag")
	}

	if req.FolderSource != nil {
		sm.reportDestination.FolderSource = *req.FolderSource
	}
	if req.Folder != nil {
		sm.reportDestination.Folder = resolveFolderPath(*req.Folder)
	}
	if req.HttpsSource != nil {
		sm.reportDestination.HttpsSource = *req.HttpsSource
	}
	if req.HTTPS != nil {
		sm.reportDestination.HTTPS = req.HTTPS
	}

	return nil
}

// GetExecutionSnapshot returns the authoritative snapshot.
func (sm *StateManager) GetExecutionSnapshot() ExecutionSnapshot {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snap := sm.snapshot
	if !sm.runStartTime.IsZero() && (sm.status == StatusRunning || sm.status == StatusRunningElevating || sm.status == StatusRunningCancelling) {
		snap.DurationMs = time.Since(sm.runStartTime).Milliseconds()
	}
	snap.Status = sm.status

	// Return deep copies of slices
	assertionsCopy := make([]AssertionSnapshot, len(snap.Assertions))
	copy(assertionsCopy, snap.Assertions)
	snap.Assertions = assertionsCopy

	logsCopy := make([]string, len(snap.Logs))
	copy(logsCopy, snap.Logs)
	snap.Logs = logsCopy

	return snap
}

// GetPlaybook returns the active playbook.
func (sm *StateManager) GetPlaybook() *playbook.Playbook {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.playbook
}

// GetLastReport returns the last generated report and whether it was cancelled.
func (sm *StateManager) GetLastReport() (*report.FinalResult, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.lastReport, sm.isCancelled
}

// HasValidationErrors returns true if current errors contain PLAYBOOK_VALIDATION_FAILED.
func (sm *StateManager) HasValidationErrors() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, err := range sm.errors {
		if err.Code == ErrCodePlaybookValidationFailed {
			return true
		}
	}
	return false
}

// GetStatus returns the current lifecycle status string.
func (sm *StateManager) GetStatus() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.status
}

// SetStatus updates the lifecycle status in a thread-safe manner.
func (sm *StateManager) SetStatus(status string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.status = status
}

// DispatchReport writes the generated report to configured destinations.
func (sm *StateManager) DispatchReport(res report.FinalResult) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 1. Folder write if active
	if sm.reportDestination.FolderSource != FolderSourceOff {
		folderDir := sm.reportDestination.Folder
		if folderDir == "" {
			folderDir = "reports"
		}
		if err := reportwriter.WriteToFolder(folderDir, res); err != nil {
			return fmt.Errorf("failed to write report to folder: %w", err)
		}
	}

	// 2. HTTPS submission if active
	if sm.reportDestination.HttpsSource == HttpsSourcePlaybook {
		if sm.playbook == nil || sm.playbook.ReportDestinationHTTPS == nil {
			return fmt.Errorf("report destination is https, but reportDestinationHttps configuration is missing")
		}
		if err := reportwriter.WriteToHTTP(sm.playbook.ReportDestinationHTTPS, res); err != nil {
			return fmt.Errorf("failed to send report to HTTPS endpoint: %w", err)
		}
	} else if sm.reportDestination.HttpsSource == HttpsSourceCustom && sm.reportDestination.HTTPS != nil {
		cfg := &playbook.ReportDestinationConfig{
			URL:               sm.reportDestination.HTTPS.URL,
			Format:            playbook.ReportFormat(sm.reportDestination.HTTPS.Format),
			SignatureSecret:   sm.reportDestination.HTTPS.Secret,
			AdditionalHeaders: sm.reportDestination.HTTPS.Headers,
		}
		if err := reportwriter.WriteToHTTP(cfg, res); err != nil {
			return fmt.Errorf("failed to send report to HTTPS endpoint: %w", err)
		}
	}

	return nil
}

