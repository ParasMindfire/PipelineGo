package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"pipeline/apps/server/repository"
	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines"
)

// ErrNoSources is returned by CreatePipeline when the spec has no input sources.
var ErrNoSources = errors.New("at least one source is required")

// ErrJobNotRunning is returned by CancelPipeline when the job isn't currently running.
var ErrJobNotRunning = errors.New("job not running")

// ErrJobNotFound re-exports the repository sentinel so callers (e.g. the
// controller) only need to depend on the service package, not the repository.
var ErrJobNotFound = repository.ErrJobNotFound

// NewPipelineService wires a repository and pipeline runner into a service.
func NewPipelineService(repo *repository.PipelineRepository, runner *pipelines.Runner) *PipelineService {
	return &PipelineService{
		repo:    repo,
		runner:  runner,
		cancels: make(map[string]context.CancelFunc),
	}
}

// CreatePipeline persists a new job and starts the pipeline asynchronously.
// It returns as soon as the job is recorded — it does not wait for the
// pipeline to finish.
func (s *PipelineService) CreatePipeline(spec models.JobSpec) (models.PipelineJob, error) {
	if len(spec.Sources) == 0 {
		return models.PipelineJob{}, ErrNoSources
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

	if err := s.repo.CreateJob(job); err != nil {
		return models.PipelineJob{}, fmt.Errorf("create job: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[job.ID] = cancel
	s.mu.Unlock()

	go func() {
		s.runner.Run(ctx, spec, job.ID)
		s.mu.Lock()
		delete(s.cancels, job.ID)
		s.mu.Unlock()
	}()

	return job, nil
}

// ListPipelines returns every job, most recently created first.
func (s *PipelineService) ListPipelines() ([]models.PipelineJob, error) {
	jobs, err := s.repo.ListJobs()
	if err != nil {
		return nil, err
	}
	if jobs == nil {
		jobs = []models.PipelineJob{}
	}
	return jobs, nil
}

// GetPipeline returns a single job by ID. Returns ErrJobNotFound if no job exists.
func (s *PipelineService) GetPipeline(id string) (*models.PipelineJob, error) {
	return s.repo.GetJob(id)
}

// GetProgress returns live metrics while the job is running, or a snapshot
// derived from the DB once it has finished.
func (s *PipelineService) GetProgress(id string) (models.ProgressMetrics, error) {
	if tracker := s.runner.GetTracker(id); tracker != nil {
		return tracker.Snapshot(), nil
	}

	job, err := s.repo.GetJob(id)
	if err != nil {
		return models.ProgressMetrics{}, err
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

	return models.ProgressMetrics{
		JobID:           id,
		Status:          job.Status.String(),
		ProcessedCount:  int64(job.RecordCount),
		ErrorCount:      int64(job.ErrorCount),
		PercentComplete: pct,
		StartTime:       startTime,
		EndTime:         job.FinishedAt,
	}, nil
}

// GetResults returns the final aggregation result, or nil if not yet computed.
func (s *PipelineService) GetResults(id string) (*models.AggregationResult, error) {
	return s.repo.GetAggregation(id)
}

// GetErrors returns every validation error recorded for a job.
func (s *PipelineService) GetErrors(id string) ([]models.ValidationError, error) {
	errs, err := s.repo.GetErrors(id)
	if err != nil {
		return nil, err
	}
	if errs == nil {
		errs = []models.ValidationError{}
	}
	return errs, nil
}

// CancelPipeline signals a running job's context to cancel. Returns
// ErrJobNotRunning if the job isn't currently running.
func (s *PipelineService) CancelPipeline(id string) error {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()

	if !ok {
		return ErrJobNotRunning
	}

	cancel()

	s.mu.Lock()
	delete(s.cancels, id)
	s.mu.Unlock()

	if err := s.repo.UpdateStatus(id, models.StatusCancelled); err != nil {
		log.Printf("CancelPipeline: update status: %v", err)
	}
	return nil
}

// DeletePipeline cancels the job if running, removes its DB rows (job_errors
// and aggregation_results cascade), and deletes its output file.
func (s *PipelineService) DeletePipeline(id string) error {
	s.mu.Lock()
	if cancel, ok := s.cancels[id]; ok {
		cancel()
		delete(s.cancels, id)
	}
	s.mu.Unlock()

	if err := s.repo.DeleteJob(id); err != nil {
		return fmt.Errorf("delete job: %w", err)
	}

	if err := os.Remove("data/output/" + id + ".json"); err != nil && !os.IsNotExist(err) {
		log.Printf("DeletePipeline: remove output file: %v", err)
	}
	return nil
}

// JobCounts tallies jobs by status across the whole job history.
func (s *PipelineService) JobCounts() (JobCounts, error) {
	byStatus, err := s.repo.CountsByStatus()
	if err != nil {
		return JobCounts{}, err
	}

	var c JobCounts
	for status, n := range byStatus {
		c.Total += n
		switch status {
		case models.StatusRunning:
			c.Running += n
		case models.StatusCompleted:
			c.Completed += n
		case models.StatusFailed, models.StatusCancelled:
			c.Failed += n
		case models.StatusPending:
			// not counted in any of the three buckets above
		}
	}
	return c, nil
}
