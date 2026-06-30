package pipelines

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/aggregation"
	"pipeline/packages/shared/pipelines/export"
	"pipeline/packages/shared/pipelines/ingestion"
	"pipeline/packages/shared/pipelines/metrics"
	"pipeline/packages/shared/pipelines/transformation"
	"pipeline/packages/shared/pipelines/validation"
)

// NewRunner creates a Runner backed by the given JobStore.
func NewRunner(s JobStore) *Runner {
	return &Runner{
		store:    s,
		exporter: export.NewExporter(s),
		trackers: make(map[string]*metrics.Tracker),
	}
}

// GetTracker returns the live tracker for a running job, or nil if finished.
func (r *Runner) GetTracker(jobID string) *metrics.Tracker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.trackers[jobID]
}

// Run executes the full pipeline for one job. Call in a dedicated goroutine —
// it blocks until the job completes, fails, or is cancelled.
func (r *Runner) Run(ctx context.Context, spec models.JobSpec, jobID string) {
	cfg := spec.Concurrency
	if cfg.ValidationWorkers == 0 {
		cfg = models.DefaultConcurrency()
	}

	tracker := metrics.NewTracker(jobID, 0)
	r.mu.Lock()
	r.trackers[jobID] = tracker
	r.mu.Unlock()

	if err := r.store.MarkStarted(jobID); err != nil {
		log.Printf("runner: mark started %s: %v", jobID, err)
	}

	progressCh := make(chan models.ProgressEvent, 500)
	go tracker.Listen(progressCh)

	// ── Stage 1: Ingestion ────────────────────────────────────────────────────
	recordsCh := ingestion.StartIngestion(ctx, spec.Sources, cfg.IngestionBufferSize)

	// ── Stage 2: Validation ───────────────────────────────────────────────────
	validCh, errorCh := validation.StartValidation(
		ctx, jobID, recordsCh, cfg.ValidationWorkers, progressCh,
	)

	// ── Stage 3: Transformation ───────────────────────────────────────────────
	transformedCh := transformation.StartTransformation(ctx, validCh, cfg.TransformWorkers)

	// ── Stage 4: Aggregation (fan-in) ─────────────────────────────────────────
	exportCh, aggCh := aggregation.StartAggregation(ctx, jobID, transformedCh)

	// ── Error collector ───────────────────────────────────────────────────────
	var errCount atomic.Int64
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for e := range errorCh {
			if err := r.store.SaveError(e); err != nil {
				log.Printf("runner: save error for job %s: %v", jobID, err)
			}
			errCount.Add(1)
		}
	}()

	// ── Stage 5: Export ───────────────────────────────────────────────────────
	outputPath := "data/output/" + jobID + ".json"
	if spec.Export.Path != "" {
		outputPath = spec.Export.Path
	}

	exportErr := r.exporter.Run(ctx, jobID, exportCh, aggCh, models.ExportConfig{
		Type: "json",
		Path: outputPath,
	})

	errWg.Wait()
	close(progressCh)

	status := models.StatusCompleted
	if ctx.Err() != nil {
		status = models.StatusCancelled
	} else if exportErr != nil {
		log.Printf("runner: export error for job %s: %v", jobID, exportErr)
		status = models.StatusFailed
	}

	snap := tracker.Snapshot()
	if err := r.store.MarkFinished(jobID, status, int(snap.ProcessedCount), int(errCount.Load())); err != nil {
		log.Printf("runner: mark finished %s: %v", jobID, err)
	}
	tracker.SetFinished(status.String())

	r.mu.Lock()
	delete(r.trackers, jobID)
	r.mu.Unlock()
}
