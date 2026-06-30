package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
