package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handleReportDestinationPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.state.CanMutate() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Cannot update report destination during active execution or submission",
			},
		})
		return
	}

	var req DestinationUpdateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "INVALID_REQUEST",
				Message: fmt.Sprintf("Failed to decode destination update payload: %v", err),
			},
		})
		return
	}

	// Validate HTTPS configuration if provided
	if req.HTTPS != nil && req.HTTPS.URL != "" {
		if !strings.HasPrefix(req.HTTPS.URL, "https://") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": AppError{
					Code:    "INVALID_DESTINATION",
					Message: "HTTPS destination URL must start with https://",
				},
			})
			return
		}
	}

	if err := s.state.UpdateDestination(req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "DESTINATION_UPDATE_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.state.GetStateResponse())
}
