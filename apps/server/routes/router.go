package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"pipeline/apps/server/controller"
	appmw "pipeline/apps/server/middleware"
)

// NewRouter builds the HTTP router with middleware and all registered routes.
// apiKey gates the mutating pipeline endpoints (create/cancel/delete) behind
// the X-API-Key header; reads stay open.
func NewRouter(c *controller.PipelineController, apiKey string) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(appmw.Logging)
	r.Use(appmw.CORS)
	r.Use(appmw.RateLimit(100, time.Minute))

	r.Get("/health", c.Health)
	r.Get("/metrics", c.Metrics)
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.UIConfig(map[string]string{"defaultModelsExpandDepth": "-1"}),
	))

	auth := appmw.APIKeyAuth(apiKey)

	r.Route("/api/{version}/pipelines", func(r chi.Router) {
		r.Use(appmw.VersionGate)
		r.With(auth).Post("/", c.CreatePipeline)
		r.Get("/", c.ListPipelines)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", c.GetPipeline)
			r.With(auth).Delete("/", c.DeletePipeline)
			r.With(auth).Patch("/cancel", c.CancelPipeline)
			r.Get("/progress", c.GetProgress)
			r.Get("/results", c.GetResults)
			r.Get("/errors", c.GetErrors)
		})
	})

	return r
}
