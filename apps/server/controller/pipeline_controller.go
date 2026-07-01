package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"pipeline/apps/server/service"
	"pipeline/packages/shared/models"
)

// NewPipelineController wires a PipelineService into a PipelineController.
func NewPipelineController(s PipelineService) *PipelineController {
	return &PipelineController{service: s}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// Health is a liveness check for the API server.
//
// @Summary      Liveness check
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (c *PipelineController) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0"})
}

// CreatePipeline decodes a JobSpec and asks the service to create and start it.
//
// @Summary      Create a pipeline job
// @Tags         pipelines
// @Accept       json
// @Produce      json
// @Param        spec  body      models.JobSpec  true  "Pipeline job specification"
// @Success      201   {object}  models.PipelineJob
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/pipelines [post]
func (c *PipelineController) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	var spec models.JobSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if err := ValidateJobSpec(spec); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	job, err := c.service.CreatePipeline(spec)
	if errors.Is(err, service.ErrNoSources) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("CreatePipeline: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create job"})
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

// ListPipelines returns every job, most recently created first.
//
// @Summary      List pipeline jobs
// @Tags         pipelines
// @Produce      json
// @Success      200  {array}   models.PipelineJob
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines [get]
func (c *PipelineController) ListPipelines(w http.ResponseWriter, r *http.Request) {
	jobs, err := c.service.ListPipelines()
	if err != nil {
		log.Printf("ListPipelines: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// GetPipeline returns a single job by ID.
//
// @Summary      Get a pipeline job
// @Tags         pipelines
// @Produce      json
// @Param        id   path      string  true  "Job ID"
// @Success      200  {object}  models.PipelineJob
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines/{id} [get]
func (c *PipelineController) GetPipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	job, err := c.service.GetPipeline(id)
	if errors.Is(err, service.ErrJobNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if err != nil {
		log.Printf("GetPipeline: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// GetProgress returns live metrics while the job is running, or a snapshot
// derived from the DB once it has finished.
//
// @Summary      Get pipeline job progress
// @Tags         pipelines
// @Produce      json
// @Param        id   path      string  true  "Job ID"
// @Success      200  {object}  models.ProgressMetrics
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines/{id}/progress [get]
func (c *PipelineController) GetProgress(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	progress, err := c.service.GetProgress(id)
	if errors.Is(err, service.ErrJobNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if err != nil {
		log.Printf("GetProgress: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

// GetResults returns the final aggregation result, or 404 if not yet computed.
//
// @Summary      Get pipeline job results
// @Tags         pipelines
// @Produce      json
// @Param        id   path      string  true  "Job ID"
// @Success      200  {object}  models.AggregationResult
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines/{id}/results [get]
func (c *PipelineController) GetResults(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	result, err := c.service.GetResults(id)
	if err != nil {
		log.Printf("GetResults: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	if result == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no results yet — job may still be running"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetErrors returns every validation error recorded for a job.
//
// @Summary      Get pipeline job validation errors
// @Tags         pipelines
// @Produce      json
// @Param        id   path      string  true  "Job ID"
// @Success      200  {array}   models.ValidationError
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines/{id}/errors [get]
func (c *PipelineController) GetErrors(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	errs, err := c.service.GetErrors(id)
	if err != nil {
		log.Printf("GetErrors: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, errs)
}

// CancelPipeline signals a running job's context to cancel. No-op (404) if
// the job isn't currently running.
//
// @Summary      Cancel a running pipeline job
// @Tags         pipelines
// @Produce      json
// @Param        id   path      string  true  "Job ID"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/pipelines/{id}/cancel [patch]
func (c *PipelineController) CancelPipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	if err := c.service.CancelPipeline(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not running"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cancellation signal sent"})
}

// DeletePipeline cancels the job if running, removes its DB rows, and deletes
// its output file.
//
// @Summary      Delete a pipeline job
// @Tags         pipelines
// @Param        id   path  string  true  "Job ID"
// @Success      204  "No Content"
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/pipelines/{id} [delete]
func (c *PipelineController) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := jobIDParam(w, r)
	if !ok {
		return
	}
	if err := c.service.DeletePipeline(id); err != nil {
		log.Printf("DeletePipeline: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Metrics returns job counts in Prometheus text exposition format.
//
// @Summary      Prometheus metrics
// @Tags         health
// @Produce      plain
// @Success      200  {string}  string  "Prometheus text exposition format"
// @Router       /metrics [get]
func (c *PipelineController) Metrics(w http.ResponseWriter, r *http.Request) {
	counts, err := c.service.JobCounts()
	if err != nil {
		log.Printf("Metrics: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	lines := []string{
		"# HELP pipeline_jobs_total Total pipeline jobs by status",
		"# TYPE pipeline_jobs_total gauge",
		fmt.Sprintf(`pipeline_jobs_total{status="running"} %d`, counts.Running),
		fmt.Sprintf(`pipeline_jobs_total{status="completed"} %d`, counts.Completed),
		fmt.Sprintf(`pipeline_jobs_total{status="failed"} %d`, counts.Failed),
		fmt.Sprintf(`pipeline_jobs_total{status="total"} %d`, counts.Total),
	}
	if _, err := fmt.Fprintln(w, strings.Join(lines, "\n")); err != nil {
		log.Printf("Metrics: write response: %v", err)
	}
}
