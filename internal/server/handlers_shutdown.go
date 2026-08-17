package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "shutting_down",
	})

	// Trigger graceful server shutdown asynchronously
	go s.TriggerShutdown()
}
