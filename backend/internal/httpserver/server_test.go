package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nightiz/pastebin-clone/backend/internal/config"
	"github.com/nightiz/pastebin-clone/backend/internal/httpserver"
	"github.com/nightiz/pastebin-clone/backend/internal/paste"
)

func setupTestApp(t *testing.T) http.Handler {
	t.Helper()
	store, err := paste.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	cfg := &config.Config{
		Port:                 8080,
		DatabasePath:         ":memory:",
		MaxPasteBytes:        524288,
		RateLimitPerMin:      1000, // High limit for test suite
		DefaultExpirySeconds: 0,
	}
	svc := paste.NewService(store, cfg.MaxPasteBytes, cfg.DefaultExpirySeconds)
	handler := paste.NewHandler(svc)
	return httpserver.New(cfg, handler)
}

func TestHTTP_CreateAndGetPaste(t *testing.T) {
	app := setupTestApp(t)

	// 1. Healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /healthz, got %d", rec.Code)
	}

	// 2. Create paste
	body := map[string]interface{}{
		"title":    "My Test Paste",
		"content":  "Hello, World!",
		"language": "plaintext",
	}
	bodyBytes, _ := json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/pastes", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var createResp paste.CreatePasteResponse
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if createResp.ID == "" || createResp.EditToken == "" {
		t.Fatalf("missing id or edit token: %+v", createResp)
	}

	// 3. Get JSON
	req = httptest.NewRequest(http.MethodGet, "/api/pastes/"+createResp.ID, nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var getResp paste.Paste
	if err := json.NewDecoder(rec.Body).Decode(&getResp); err != nil {
		t.Fatalf("failed to decode paste: %v", err)
	}
	if getResp.Title != "My Test Paste" || getResp.Content != "Hello, World!" {
		t.Fatalf("unexpected paste content: %+v", getResp)
	}

	// 4. Get Raw text/plain
	req = httptest.NewRequest(http.MethodGet, "/api/pastes/"+createResp.ID+"/raw", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for raw, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("expected text/plain; charset=utf-8, got %q", ct)
	}
	if rec.Body.String() != "Hello, World!" {
		t.Fatalf("expected raw content 'Hello, World!', got %q", rec.Body.String())
	}

	// 5. Delete with token
	req = httptest.NewRequest(http.MethodDelete, "/api/pastes/"+createResp.ID, nil)
	req.Header.Set("X-Edit-Token", createResp.EditToken)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// 6. Verify 404 after delete
	req = httptest.NewRequest(http.MethodGet, "/api/pastes/"+createResp.ID, nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found after delete, got %d", rec.Code)
	}
}

func TestHTTP_MaxBodySize(t *testing.T) {
	store, err := paste.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	cfg := &config.Config{
		Port:                 8080,
		DatabasePath:         ":memory:",
		MaxPasteBytes:        50, // Small 50-byte max body limit
		RateLimitPerMin:      1000,
		DefaultExpirySeconds: 0,
	}
	svc := paste.NewService(store, cfg.MaxPasteBytes, cfg.DefaultExpirySeconds)
	handler := paste.NewHandler(svc)
	app := httpserver.New(cfg, handler)

	oversized := map[string]interface{}{
		"content": string(make([]byte, 200)),
	}
	bodyBytes, _ := json.Marshal(oversized)

	req := httptest.NewRequest(http.MethodPost, "/api/pastes", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 Payload Too Large, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestHTTP_RateLimiter(t *testing.T) {
	store, err := paste.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	cfg := &config.Config{
		Port:                 8080,
		DatabasePath:         ":memory:",
		MaxPasteBytes:        524288,
		RateLimitPerMin:      2, // Limit to 2 write requests
		DefaultExpirySeconds: 0,
	}
	svc := paste.NewService(store, cfg.MaxPasteBytes, cfg.DefaultExpirySeconds)
	handler := paste.NewHandler(svc)
	app := httpserver.New(cfg, handler)

	bodyBytes, _ := json.Marshal(map[string]interface{}{"content": "test"})

	sendWrite := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/pastes", bytes.NewReader(bodyBytes))
		req.RemoteAddr = "192.0.2.1:12345"
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		_, _ = io.ReadAll(rec.Body)
		return rec.Code
	}

	// First two should succeed (201)
	if code := sendWrite(); code != http.StatusCreated {
		t.Fatalf("request 1 failed with %d", code)
	}
	if code := sendWrite(); code != http.StatusCreated {
		t.Fatalf("request 2 failed with %d", code)
	}

	// Third request should be rate-limited (429)
	if code := sendWrite(); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", code)
	}
}
