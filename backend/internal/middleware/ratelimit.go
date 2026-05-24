package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count     int
	windowEnd time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]visitor
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 120
	}
	if window <= 0 {
		window = time.Minute
	}

	rl := &RateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]visitor),
	}

	go rl.cleanupExpiredClients(2 * window)
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r), time.Now()) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "RATE_LIMITED",
					"message": "Too many requests",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.clients[key]
	if !ok || now.After(v.windowEnd) {
		rl.clients[key] = visitor{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}

	if v.count >= rl.limit {
		return false
	}

	v.count++
	rl.clients[key] = v
	return true
}

func (rl *RateLimiter) cleanupExpiredClients(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for now := range ticker.C {
		rl.mu.Lock()
		for key, v := range rl.clients {
			if now.After(v.windowEnd) {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
