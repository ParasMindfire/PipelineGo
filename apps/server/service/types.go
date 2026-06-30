package service

import (
	"context"
	"sync"

	"pipeline/apps/server/repository"
	"pipeline/packages/shared/pipelines"
)

// PipelineService holds the business logic for creating, tracking, and
// cancelling pipeline jobs. It owns the in-memory map of cancel functions for
// running jobs — that's request-lifetime state, not something the repository
// or the pipeline engine should know about.
type PipelineService struct {
	repo    *repository.PipelineRepository
	runner  *pipelines.Runner
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
