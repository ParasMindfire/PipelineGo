package service

import (
	"context"
	"sync"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/metrics"
)

// PipelineRepo is the persistence surface the service needs. Defined as an
// interface here so tests can inject a mock without touching a real database.
type PipelineRepo interface {
	CreateJob(models.PipelineJob) error
	GetJob(id string) (*models.PipelineJob, error)
	ListJobs() ([]models.PipelineJob, error)
	DeleteJob(id string) error
	MarkStarted(id string) error
	MarkFinished(id string, status models.JobStatus, recordCount, errorCount int) error
	UpdateStatus(id string, status models.JobStatus) error
	SaveError(e models.ValidationError) error
	GetErrors(jobID string) ([]models.ValidationError, error)
	SaveAggregation(r models.AggregationResult) error
	GetAggregation(jobID string) (*models.AggregationResult, error)
	CountsByStatus() (map[models.JobStatus]int, error)
}

// JobRunner is the concurrency surface the service needs from the pipeline
// runner. Defined as an interface so tests can inject a no-op mock runner.
type JobRunner interface {
	Run(ctx context.Context, spec models.JobSpec, jobID string)
	GetTracker(jobID string) *metrics.Tracker
}

// PipelineService holds the business logic for creating, tracking, and
// cancelling pipeline jobs. It owns the in-memory map of cancel functions for
// running jobs — that's request-lifetime state, not something the repository
// or the pipeline engine should know about.
type PipelineService struct {
	repo    PipelineRepo
	runner  JobRunner
	cancels map[string]context.CancelFunc
	mu      sync.Mutex
}

// JobCounts summarizes job statuses for the Prometheus metrics endpoint.
type JobCounts struct {
	Running   int
	Completed int
	Failed    int
	Total     int
}
