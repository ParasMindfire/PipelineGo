package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"pipeline/internal/models"
	"pipeline/internal/pipeline"
	"pipeline/internal/store"
)

// Handler holds every dependency the HTTP layer needs.
// cancels maps a running job ID to the CancelFunc that stops it.
type Handler struct {
	store   *store.Store
	runner  *pipeline.Runner
	cancels map[string]context.CancelFunc
	mu      sync.Mutex
}

// NewHandler wires the store and pipeline runner into a Handler.
func NewHandler(s *store.Store, r *pipeline.Runner) *Handler {
	return &Handler{
		store:   s,
		runner:  r,
		cancels: make(map[string]context.CancelFunc),
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// Health is a liveness check for the API server.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0"})
}

// CreatePipeline decodes a JobSpec, persists a new job, and starts the
// pipeline asynchronously. Responds 201 with the created job immediately —
// it does not wait for the pipeline to finish.
func (h *Handler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	var spec models.JobSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if len(spec.Sources) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one source is required"})
		return
	}

	if spec.Concurrency.ValidationWorkers == 0 {
		spec.Concurrency = models.DefaultConcurrency()
	}

	job := models.PipelineJob{
		ID:        uuid.New().String(),
		Status:    models.StatusPending,
		Spec:      spec,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateJob(job); err != nil {
		log.Printf("CreatePipeline: create job: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create job"})
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancels[job.ID] = cancel
	h.mu.Unlock()

	go func() {
		h.runner.Run(ctx, spec, job.ID)
		// job finished (completed, failed, or cancelled) — stop tracking its CancelFunc
		h.mu.Lock()
		delete(h.cancels, job.ID)
		h.mu.Unlock()
	}()

	writeJSON(w, http.StatusCreated, job)
}

// ListPipelines returns every job, most recently created first.
func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.store.ListJobs()
	if err != nil {
		log.Printf("ListPipelines: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	if jobs == nil {
		jobs = []models.PipelineJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

// GetPipeline returns a single job by ID.
func (h *Handler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.store.GetJob(id)
	if errors.Is(err, store.ErrJobNotFound) {
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
func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if tracker := h.runner.GetTracker(id); tracker != nil {
		writeJSON(w, http.StatusOK, tracker.Snapshot())
		return
	}

	job, err := h.store.GetJob(id)
	if errors.Is(err, store.ErrJobNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if err != nil {
		log.Printf("GetProgress: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}

	pct := 0.0
	if job.Status == models.StatusCompleted {
		pct = 100.0
	}

	// StartedAt is nil if the job hasn't been picked up by the runner goroutine
	// yet (a brief window right after creation) — fall back to CreatedAt.
	startTime := job.CreatedAt
	if job.StartedAt != nil {
		startTime = *job.StartedAt
	}

	writeJSON(w, http.StatusOK, models.ProgressMetrics{
		JobID:           id,
		Status:          job.Status.String(),
		ProcessedCount:  int64(job.RecordCount),
		ErrorCount:      int64(job.ErrorCount),
		PercentComplete: pct,
		StartTime:       startTime,
		EndTime:         job.FinishedAt,
	})
}

// GetResults returns the final aggregation result, or 404 if not yet computed.
func (h *Handler) GetResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.store.GetAggregation(id)
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
func (h *Handler) GetErrors(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	errs, err := h.store.GetErrors(id)
	if err != nil {
		log.Printf("GetErrors: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	if errs == nil {
		errs = []models.ValidationError{}
	}
	writeJSON(w, http.StatusOK, errs)
}

// CancelPipeline signals a running job's context to cancel. No-op (404) if
// the job isn't currently running.
func (h *Handler) CancelPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	h.mu.Lock()
	cancel, ok := h.cancels[id]
	h.mu.Unlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not running"})
		return
	}

	cancel()

	h.mu.Lock()
	delete(h.cancels, id)
	h.mu.Unlock()

	if err := h.store.UpdateStatus(id, models.StatusCancelled); err != nil {
		log.Printf("CancelPipeline: update status: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cancellation signal sent"})
}

// DeletePipeline cancels the job if running, removes its DB rows
// (job_errors and aggregation_results cascade), and deletes its output file.
func (h *Handler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	h.mu.Lock()
	if cancel, ok := h.cancels[id]; ok {
		cancel()
		delete(h.cancels, id)
	}
	h.mu.Unlock()

	if err := h.store.DeleteJob(id); err != nil {
		log.Printf("DeletePipeline: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}

	if err := os.Remove("data/output/" + id + ".json"); err != nil && !os.IsNotExist(err) {
		log.Printf("DeletePipeline: remove output file: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Metrics returns job counts in Prometheus text exposition format.
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.store.ListJobs()
	if err != nil {
		log.Printf("Metrics: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var running, completed, failed int
	for i := range jobs {
		switch jobs[i].Status {
		case models.StatusRunning:
			running++
		case models.StatusCompleted:
			completed++
		case models.StatusFailed, models.StatusCancelled:
			failed++
		case models.StatusPending:
			// not counted in any of the three buckets above
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	lines := []string{
		"# HELP pipeline_jobs_total Total pipeline jobs by status",
		"# TYPE pipeline_jobs_total gauge",
		fmt.Sprintf(`pipeline_jobs_total{status="running"} %d`, running),
		fmt.Sprintf(`pipeline_jobs_total{status="completed"} %d`, completed),
		fmt.Sprintf(`pipeline_jobs_total{status="failed"} %d`, failed),
		fmt.Sprintf(`pipeline_jobs_total{status="total"} %d`, len(jobs)),
	}
	if _, err := fmt.Fprintln(w, strings.Join(lines, "\n")); err != nil {
		log.Printf("Metrics: write response: %v", err)
	}
}
