package pipelines

import (
	"sync"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/export"
	"pipeline/packages/shared/pipelines/metrics"
)

// JobStore is the persistence surface the Runner needs. It's an interface
// (not the concrete repository) so this package never depends on apps/server —
// the concrete implementation is injected by whichever app wires the Runner up.
type JobStore interface {
	MarkStarted(id string) error
	SaveError(e models.ValidationError) error
	MarkFinished(id string, status models.JobStatus, recordCount, errorCount int) error
	SaveAggregation(r models.AggregationResult) error
}

// Runner wires all pipeline stages together and tracks active jobs.
type Runner struct {
	store    JobStore
	exporter *export.Exporter
	trackers map[string]*metrics.Tracker
	mu       sync.RWMutex
}
