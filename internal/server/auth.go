package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	SessionCookieName = "crobe_session"
)

// GenerateToken generates a cryptographically random 32-byte hex token string.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SecurityHeadersMiddleware sets strict CSP, no-store cache headers, and basic security headers.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; connect-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware enforces token authentication via crobe_session cookie or Authorization header.
func AuthMiddleware(serverToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serverToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := extractAuthToken(r)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(serverToken)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": AppError{
					Code:    "UNAUTHORIZED",
					Message: "Authentication required or invalid session token",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractAuthToken(r *http.Request) string {
	// 1. Check Cookie
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Check Authorization Header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	// 3. Check Query parameter
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		return qToken
	}

	return ""
}
