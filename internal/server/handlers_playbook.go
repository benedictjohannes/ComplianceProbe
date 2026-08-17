package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/benedictjohannes/crobe/playbook"
	"gopkg.in/yaml.v3"
)

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := s.state.GetStateResponse()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePlaybookUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.state.CanMutate() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Cannot upload playbook during active execution or submission",
			},
		})
		return
	}

	// Limit upload size to 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		s.state.SetLoadError(ErrCodePlaybookParseFailed, fmt.Sprintf("Failed to parse multipart upload: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.state.SetLoadError(ErrCodePlaybookParseFailed, "Missing 'file' field in multipart upload", nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		s.state.SetLoadError(ErrCodePlaybookParseFailed, fmt.Sprintf("Failed to read uploaded file: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	filename := strings.ToLower(header.Filename)
	isJSON := strings.HasSuffix(filename, ".json")

	var pb playbook.Playbook
	if isJSON {
		err = json.Unmarshal(data, &pb)
	} else {
		err = yaml.Unmarshal(data, &pb)
	}

	if err != nil {
		s.state.SetLoadError(ErrCodePlaybookParseFailed, fmt.Sprintf("Failed to parse playbook: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate playbook as Agent
	valErrors := pb.Validate(true)
	s.state.SetPlaybook(&pb, data, valErrors)
	resp := s.state.GetStateResponse()
	if s.broker != nil {
		s.broker.Broadcast("state_change", "", resp)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// RemotePlaybookClient allows overriding the HTTP client for remote playbook fetches in tests.
var RemotePlaybookClient *http.Client

func (s *Server) handlePlaybookRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.state.CanMutate() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: "Cannot fetch remote playbook during active execution or submission",
			},
		})
		return
	}

	var req RemotePlaybookRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, fmt.Sprintf("Invalid JSON request body: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if !strings.HasPrefix(req.URL, "https://") {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, "Remote playbook URL must use HTTPS", nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	client := RemotePlaybookClient
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after 5 redirects")
				}
				if req.URL.Scheme != "https" {
					return fmt.Errorf("insecure redirect to non-HTTPS scheme: %s", req.URL.String())
				}
				return nil
			},
		}
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, req.URL, nil)
	if err != nil {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, fmt.Sprintf("Failed to create HTTP request: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	respClient, err := client.Do(httpReq)
	if err != nil {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, fmt.Sprintf("Failed to fetch remote playbook: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	defer respClient.Body.Close()

	if respClient.StatusCode != http.StatusOK {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, fmt.Sprintf("Remote server responded with status %d", respClient.StatusCode), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Limit download size to 5 MB
	data, err := io.ReadAll(io.LimitReader(respClient.Body, 5<<20))
	if err != nil {
		s.state.SetLoadError(ErrCodeRemoteFetchFailed, fmt.Sprintf("Failed to read remote playbook: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	contentType := respClient.Header.Get("Content-Type")
	isJSON := strings.HasPrefix(strings.ToLower(contentType), "application/json")

	var pb playbook.Playbook
	if isJSON {
		err = json.Unmarshal(data, &pb)
	} else {
		err = yaml.Unmarshal(data, &pb)
	}

	if err != nil {
		s.state.SetLoadError(ErrCodePlaybookParseFailed, fmt.Sprintf("Failed to parse remote playbook: %v", err), nil)
		resp := s.state.GetStateResponse()
		if s.broker != nil {
			s.broker.Broadcast("state_change", "", resp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate as Agent
	valErrors := pb.Validate(true)
	s.state.SetPlaybook(&pb, data, valErrors)
	resp := s.state.GetStateResponse()
	if s.broker != nil {
		s.broker.Broadcast("state_change", "", resp)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePlaybookGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	inspection, err := s.state.GetPlaybookInspection()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    "NO_PLAYBOOK",
				Message: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(inspection)
}

func (s *Server) handlePlaybookDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.state.UnloadPlaybook(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": AppError{
				Code:    ErrCodeConflict,
				Message: err.Error(),
			},
		})
		return
	}

	resp := s.state.GetStateResponse()
	if s.broker != nil {
		s.broker.Broadcast("state_change", "", resp)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

