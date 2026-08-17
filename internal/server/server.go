package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/benedictjohannes/crobe/internal/webui"
)

// Config contains configuration options for the embedded UI server.
type Config struct {
	Host                string
	Port                int
	Token               string
	CLIFolder           string
	NoOpen              bool
	StartupGracePeriod  time.Duration
	InactivityTimeout   time.Duration
	DisableAutoShutdown bool
}

// Server is the embedded HTTP server for Compliance Probe Web UI.
type Server struct {
	config       Config
	httpServer   *http.Server
	listener     net.Listener
	state        *StateManager
	broker       *EventBroker
	lifecycle    *LifecycleManager
	shutdownChan chan struct{}
	shutdownOnce sync.Once
}

// NewServer initializes a new Server instance.
func NewServer(cfg Config) (*Server, error) {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Token == "" {
		token, err := GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate bootstrap token: %w", err)
		}
		cfg.Token = token
	}

	broker := NewEventBroker()
	state := NewStateManager(cfg.CLIFolder)

	s := &Server{
		config:       cfg,
		state:        state,
		broker:       broker,
		shutdownChan: make(chan struct{}),
	}

	lifecycle := NewLifecycleManager(state, LifecycleConfig{
		StartupGracePeriod:  cfg.StartupGracePeriod,
		InactivityTimeout:   cfg.InactivityTimeout,
		DisableAutoShutdown: cfg.DisableAutoShutdown,
	}, func() {
		s.TriggerShutdown()
	})
	s.lifecycle = lifecycle

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Handler:      SecurityHeadersMiddleware(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s, nil
}


func (s *Server) registerRoutes(mux *http.ServeMux) {
	// API routes protected by AuthMiddleware
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/state", s.handleState)
	apiMux.HandleFunc("/api/playbook/upload", s.handlePlaybookUpload)
	apiMux.HandleFunc("/api/playbook/remote", s.handlePlaybookRemote)
	apiMux.HandleFunc("/api/playbook", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handlePlaybookGet(w, r)
		case http.MethodDelete:
			s.handlePlaybookDelete(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/report/destination", s.handleReportDestinationPut)
	apiMux.HandleFunc("/api/run", s.handleRun)
	apiMux.HandleFunc("/api/execution/cancel", s.handleCancel)
	apiMux.HandleFunc("/api/execution", s.handleExecutionGet)
	apiMux.HandleFunc("/api/report", s.handleReportGet)
	apiMux.HandleFunc("/api/report/md", s.handleReportMDGet)
	apiMux.HandleFunc("/api/report/log", s.handleReportLogGet)
	apiMux.HandleFunc("/api/report/download", s.handleReportDownload)
	apiMux.HandleFunc("/api/report/remote-submit", s.handleReportRemoteSubmit)
	apiMux.HandleFunc("/api/events", s.handleEvents)
	apiMux.HandleFunc("/api/shutdown", s.handleShutdown)

	// Mount protected API routes
	mux.Handle("/api/", AuthMiddleware(s.config.Token, apiMux))

	// Static & SPA routes
	fs, err := webui.GetFileSystem()
	var fileServer http.Handler
	if err == nil {
		fileServer = http.FileServer(fs)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Handle Bootstrap Token Redirect: GET /?token=...
		if token := r.URL.Query().Get("token"); token != "" {
			if subtle.ConstantTimeCompare([]byte(token), []byte(s.config.Token)) == 1 {
				http.SetCookie(w, &http.Cookie{
					Name:     SessionCookieName,
					Value:    s.config.Token,
					Path:     "/",
					SameSite: http.SameSiteStrictMode,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// 2. Serve static files or fallback to index.html for SPA
		path := r.URL.Path
		if path == "/" || path == "/index.html" || !strings.Contains(path, ".") {
			indexData, err := webui.GetIndexHTML()
			if err != nil {
				http.Error(w, "Embedded index.html not found", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexData)
			return
		}

		if fileServer != nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

// Start binds the listener and starts serving HTTP requests.
func (s *Server) Start() error {
	if s.listener != nil {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	go func() {
		_ = s.httpServer.Serve(s.listener)
	}()

	return nil
}

// ListeningAddr returns the actual network address the server is listening on.
func (s *Server) ListeningAddr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// URL returns the full reachable HTTP URL with bootstrap token.
func (s *Server) URL() string {
	addr := s.ListeningAddr()
	// If bound to 0.0.0.0, present 127.0.0.1 for local browser reachability
	if strings.HasPrefix(addr, "0.0.0.0:") {
		addr = "127.0.0.1" + addr[7:]
	} else if strings.HasPrefix(addr, "[::]:") {
		addr = "127.0.0.1" + addr[4:]
	}
	return fmt.Sprintf("http://%s/?token=%s", addr, s.config.Token)
}

// StateManager returns the underlying StateManager instance.
func (s *Server) StateManager() *StateManager {
	return s.state
}

// EventBroker returns the underlying EventBroker instance.
func (s *Server) EventBroker() *EventBroker {
	return s.broker
}

// LifecycleManager returns the underlying LifecycleManager instance.
func (s *Server) LifecycleManager() *LifecycleManager {
	return s.lifecycle
}

// Token returns the configured bootstrap token.
func (s *Server) Token() string {
	return s.config.Token
}

// TriggerShutdown initiates asynchronous graceful shutdown.
func (s *Server) TriggerShutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownChan)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
}

// Shutdown gracefully terminates the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.lifecycle != nil {
		s.lifecycle.Stop()
	}

	s.state.mu.Lock()
	if s.state.activeRunCancel != nil {
		s.state.activeRunCancel()
	}
	s.state.mu.Unlock()

	return s.httpServer.Shutdown(ctx)
}

// Close immediately terminates the HTTP server.
func (s *Server) Close() error {
	if s.lifecycle != nil {
		s.lifecycle.Stop()
	}

	s.state.mu.Lock()
	if s.state.activeRunCancel != nil {
		s.state.activeRunCancel()
	}
	s.state.mu.Unlock()

	return s.httpServer.Close()
}

// ShutdownChan returns the channel closed when shutdown is triggered.
func (s *Server) ShutdownChan() <-chan struct{} {
	return s.shutdownChan
}

