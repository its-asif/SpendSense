package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Recoverer catches panics and returns a consistent 500 response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered path=%s method=%s panic=%v", r.URL.Path, r.Method, rec)
				log.Printf("stack trace:\n%s", debug.Stack())

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestLogger logs incoming requests with status and duration.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("method=%s path=%s status=%d duration=%s remote=%s", r.Method, r.URL.Path, rec.status, time.Since(start), r.RemoteAddr)
	})
}
