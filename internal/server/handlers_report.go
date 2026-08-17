package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
)

func (s *Server) handleReportGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rep, isCancelled := s.state.GetLastReport()
	if rep == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_REPORT",
				Message: "No report available. Run a playbook first.",
			},
		})
		return
	}

	if isCancelled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Structured JSON report is unavailable for cancelled runs",
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep.Structured)
}

func (s *Server) handleReportMDGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rep, _ := s.state.GetLastReport()
	if rep == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_REPORT",
				Message: "No markdown report available. Run a playbook first.",
			},
		})
		return
	}

	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\"report.md\"")
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(rep.Markdown))
}

func (s *Server) handleReportLogGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rep, _ := s.state.GetLastReport()
	if rep == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_REPORT",
				Message: "No execution log available. Run a playbook first.",
			},
		})
		return
	}

	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\"report.log\"")
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(rep.Log))
}

func (s *Server) handleReportDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rep, isCancelled := s.state.GetLastReport()
	if rep == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_REPORT",
				Message: "No report bundle available to download. Run a playbook first.",
			},
		})
		return
	}

	if isCancelled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Report archive download is unavailable for cancelled runs",
			},
		})
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "zip"
	}

	data, contentType, filename, err := BuildReportArchive(*rep, format)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "ARCHIVE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func (s *Server) handleReportRemoteSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.state.mu.Lock()
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
				Message: "Cannot submit remote report while execution or submission is in progress",
			},
		})
		return
	}

	if s.state.lastReport == nil || s.state.isCancelled {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Cannot submit remote report: no valid completed report available",
			},
		})
		return
	}

	// Resolve destination configuration
	var httpsConfig *playbook.ReportDestinationConfig
	if s.state.reportDestination.HttpsSource == HttpsSourceCustom && s.state.reportDestination.HTTPS != nil {
		httpsConfig = &playbook.ReportDestinationConfig{
			URL:               s.state.reportDestination.HTTPS.URL,
			Format:            playbook.ReportFormat(s.state.reportDestination.HTTPS.Format),
			SignatureSecret:   s.state.reportDestination.HTTPS.Secret,
			AdditionalHeaders: s.state.reportDestination.HTTPS.Headers,
		}
	} else if s.state.reportDestination.HttpsSource == HttpsSourcePlaybook && s.state.playbook != nil && s.state.playbook.ReportDestinationHTTPS != nil {
		httpsConfig = s.state.playbook.ReportDestinationHTTPS
	}

	if httpsConfig == nil || httpsConfig.URL == "" {
		s.state.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_DESTINATION",
				Message: "No remote HTTPS destination configured",
			},
		})
		return
	}

	rep := *s.state.lastReport
	s.state.status = StatusCompletedSubmitting
	runID := s.state.activeRunID
	stateResp := s.state.getStateResponseLocked()
	s.state.mu.Unlock()

	if s.broker != nil {
		s.broker.Broadcast("state_change", runID, stateResp)
	}

	// Perform Remote Write
	err := reportwriter.WriteToHTTP(httpsConfig, rep)

	s.state.mu.Lock()
	if err != nil {
		s.state.status = StatusCompletedConfirmingSubmission
		s.state.errors = append(s.state.errors, AppError{
			Code:    ErrCodeRemoteSubmissionFailed,
			Message: fmt.Sprintf("Remote submission failed: %v", err),
		})
		resp := s.state.getStateResponseLocked()
		s.state.mu.Unlock()

		if s.broker != nil {
			s.broker.Broadcast("state_change", runID, resp)
		}
		if s.lifecycle != nil {
			s.lifecycle.OnExecutionStateChange()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Submission Succeeded
	s.state.status = StatusCompleted
	// Clear previous submission errors
	var remainingErrors []AppError
	for _, e := range s.state.errors {
		if e.Code != ErrCodeRemoteSubmissionFailed {
			remainingErrors = append(remainingErrors, e)
		}
	}
	s.state.errors = remainingErrors
	resp := s.state.getStateResponseLocked()
	s.state.mu.Unlock()

	if s.broker != nil {
		s.broker.Broadcast("state_change", runID, resp)
	}
	if s.lifecycle != nil {
		s.lifecycle.OnExecutionStateChange()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
