package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"
)

// Logging logs method, path, and duration for every request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS allows cross-origin requests from any origin.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// supportedAPIVersions lists the {version} path segments VersionGate accepts.
var supportedAPIVersions = map[string]bool{"v1": true}

// VersionGate reads the {version} URL param set by a chi route like
// "/api/{version}/pipelines" and rejects anything not in supportedAPIVersions
// before the request reaches a handler, so adding/dropping a version is a
// one-line change here instead of duplicating route trees per version.
func VersionGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := chi.URLParam(r, "version")
		if !supportedAPIVersions[version] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported API version: " + version})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimit caps each client IP to maxRequests per window using a token
// bucket, returning 429 once exceeded. IP is taken from RemoteAddr rather
// than X-Forwarded-For: this server isn't documented as running behind a
// trusted reverse proxy, and trusting a client-supplied header here would
// let the header itself be spoofed to dodge the limit entirely.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var mu sync.Mutex
	clients := make(map[string]*entry)

	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for ip, e := range clients {
				if time.Since(e.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				ip = host
			}

			mu.Lock()
			e, ok := clients[ip]
			if !ok {
				e = &entry{limiter: rate.NewLimiter(rate.Every(window/time.Duration(maxRequests)), maxRequests)}
				clients[ip] = e
			}
			e.lastSeen = time.Now()
			limiter := e.limiter
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// APIKeyAuth requires the X-API-Key header to match key, comparing in
// constant time so the check doesn't leak key contents through timing.
func APIKeyAuth(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-API-Key")
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing or invalid api key"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
