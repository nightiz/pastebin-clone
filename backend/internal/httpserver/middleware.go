package httpserver

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nightiz/pastebin-clone/backend/internal/apierr"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Recoverer catches panics and returns 500 INTERNAL
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				apierr.WriteErr(w, apierr.Internal("an internal server error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs HTTP method, path, status, and latency via slog
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", duration,
		)
	})
}

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter implements a per-IP token bucket on write endpoints (POST, DELETE).
// ponytail: in-memory per-IP map with periodic prune. If multi-instance scale is needed, replace with Redis.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientBucket
	capacity float64
	rate     float64 // tokens per second
}

func NewRateLimiter(limitPerMin int) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*clientBucket),
		capacity: float64(limitPerMin),
		rate:     float64(limitPerMin) / 60.0,
	}

	// Periodic cleanup of stale client buckets every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, b := range rl.clients {
				if now.Sub(b.lastRefill) > 10*time.Minute {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only rate-limit write endpoints
		if r.Method == http.MethodPost || r.Method == http.MethodDelete {
			ip := getClientIP(r)
			if !rl.allow(ip) {
				apierr.WriteErr(w, apierr.RateLimited("rate limit exceeded, please try again shortly"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientBucket{
			tokens:     rl.capacity - 1.0,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

// MaxBodySize wraps request body with http.MaxBytesReader
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
