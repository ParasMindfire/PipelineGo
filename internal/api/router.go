package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the HTTP router with middleware and all registered routes.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	r.Get("/health", h.Health)
	r.Get("/metrics", h.Metrics)

	r.Route("/api/v1/pipelines", func(r chi.Router) {
		r.Post("/", h.CreatePipeline)
		r.Get("/", h.ListPipelines)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetPipeline)
			r.Delete("/", h.DeletePipeline)
			r.Patch("/cancel", h.CancelPipeline)
			r.Get("/progress", h.GetProgress)
			r.Get("/results", h.GetResults)
			r.Get("/errors", h.GetErrors)
		})
	})

	return r
}
