package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VakZok/chirpy/internal/auth"
	"github.com/google/uuid"
)

// These tests exercise the authentication logic in createChirpHandler that
// runs before any database access (cfg.db is intentionally left nil, since
// a valid request never reaches it in these scenarios).

func TestCreateChirpHandler_MissingAuthHeader(t *testing.T) {
	cfg := &Config{jwtSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodPost, "/api/chirps", strings.NewReader(`{"body":"hello"}`))
	w := httptest.NewRecorder()

	cfg.createChirpHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestCreateChirpHandler_InvalidToken(t *testing.T) {
	cfg := &Config{jwtSecret: "test-secret"}

	req := httptest.NewRequest(http.MethodPost, "/api/chirps", strings.NewReader(`{"body":"hello"}`))
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	w := httptest.NewRecorder()

	cfg.createChirpHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestCreateChirpHandler_TokenSignedWithWrongSecret(t *testing.T) {
	cfg := &Config{jwtSecret: "correct-secret"}

	token, err := auth.MakeJWT(uuid.New(), "wrong-secret", time.Hour)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chirps", strings.NewReader(`{"body":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	cfg.createChirpHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestCreateChirpHandler_ExpiredToken(t *testing.T) {
	cfg := &Config{jwtSecret: "test-secret"}

	token, err := auth.MakeJWT(uuid.New(), "test-secret", -time.Hour)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chirps", strings.NewReader(`{"body":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	cfg.createChirpHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
