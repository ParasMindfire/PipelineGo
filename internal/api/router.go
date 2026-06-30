package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the HTTP router with middleware and all registered routes.
// Routes beyond /health are added incrementally as their handlers are implemented.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	r.Get("/health", h.Health)

	r.Route("/api/v1/pipelines", func(r chi.Router) {
		r.Post("/", h.CreatePipeline)
		r.Get("/", h.ListPipelines)
	})

	return r
}
