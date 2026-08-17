package server

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	tok1, err1 := GenerateToken()
	if err1 != nil {
		t.Fatalf("GenerateToken failed: %v", err1)
	}
	tok2, err2 := GenerateToken()
	if err2 != nil {
		t.Fatalf("GenerateToken failed: %v", err2)
	}

	if len(tok1) != 64 {
		t.Errorf("expected 64 hex characters (32 bytes), got length %d", len(tok1))
	}
	if tok1 == tok2 {
		t.Errorf("expected randomly generated tokens to be unique, got duplicate: %s", tok1)
	}

	decoded, err := hex.DecodeString(tok1)
	if err != nil || len(decoded) != 32 {
		t.Errorf("token is not valid hex-encoded 32 bytes: %v", err)
	}
}

func TestExtractAuthToken(t *testing.T) {
	// 1. Authorization header: Bearer token
	req1 := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req1.Header.Set("Authorization", "Bearer my-secret-token")
	if tok := extractAuthToken(req1); tok != "my-secret-token" {
		t.Errorf("expected 'my-secret-token', got %q", tok)
	}

	// 2. Cookie session
	req2 := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	req2.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: "cookie-token",
	})
	if tok := extractAuthToken(req2); tok != "cookie-token" {
		t.Errorf("expected 'cookie-token', got %q", tok)
	}

	// 3. Query token
	req3 := httptest.NewRequest(http.MethodGet, "/api/state?token=query-token", nil)
	if tok := extractAuthToken(req3); tok != "query-token" {
		t.Errorf("expected 'query-token', got %q", tok)
	}

	// 4. No auth provided
	req4 := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	if tok := extractAuthToken(req4); tok != "" {
		t.Errorf("expected empty token, got %q", tok)
	}
}
