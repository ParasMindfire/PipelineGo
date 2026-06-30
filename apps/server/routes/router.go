package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"pipeline/apps/server/controller"
	appmw "pipeline/apps/server/middleware"
)

// NewRouter builds the HTTP router with middleware and all registered routes.
func NewRouter(c *controller.PipelineController) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(appmw.Logging)
	r.Use(appmw.CORS)

	r.Get("/health", c.Health)
	r.Get("/metrics", c.Metrics)
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.UIConfig(map[string]string{"defaultModelsExpandDepth": "-1"}),
	))

	r.Route("/api/{version}/pipelines", func(r chi.Router) {
		r.Use(appmw.VersionGate)
		r.Post("/", c.CreatePipeline)
		r.Get("/", c.ListPipelines)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", c.GetPipeline)
			r.Delete("/", c.DeletePipeline)
			r.Patch("/cancel", c.CancelPipeline)
			r.Get("/progress", c.GetProgress)
			r.Get("/results", c.GetResults)
			r.Get("/errors", c.GetErrors)
		})
	})

	return r
}
