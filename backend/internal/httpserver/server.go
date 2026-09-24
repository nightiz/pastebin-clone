package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/nightiz/pastebin-clone/backend/internal/config"
	"github.com/nightiz/pastebin-clone/backend/internal/paste"
)

// New creates and configures the root http.Handler with routing and middlewares.
func New(cfg *config.Config, pasteHandler *paste.Handler) http.Handler {
	mux := http.NewServeMux()

	// Paste endpoints
	mux.HandleFunc("POST /api/pastes", pasteHandler.Create)
	mux.HandleFunc("GET /api/pastes/{id}", pasteHandler.Get)
	mux.HandleFunc("GET /api/pastes/{id}/raw", pasteHandler.GetRaw)
	mux.HandleFunc("DELETE /api/pastes/{id}", pasteHandler.Delete)

	// Liveness check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Middleware chain applied in exact order:
	// 1. recoverer
	// 2. requestLogger
	// 3. rateLimiter
	// 4. maxBodySize
	rl := NewRateLimiter(cfg.RateLimitPerMin)

	var handler http.Handler = mux
	handler = MaxBodySize(cfg.MaxPasteBytes)(handler)
	handler = rl.Middleware(handler)
	handler = RequestLogger(handler)
	handler = Recoverer(handler)

	return handler
}
