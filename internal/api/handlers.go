package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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
