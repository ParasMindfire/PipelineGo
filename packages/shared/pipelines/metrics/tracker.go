package metrics

import (
	"time"

	"pipeline/packages/shared/models"
)

// NewTracker creates a tracker for a job. Pass totalExpected=0 when the total
// record count isn't known upfront — PercentComplete will return -1 in that case.
func NewTracker(jobID string, totalExpected int64) *Tracker {
	return &Tracker{
		jobID:         jobID,
		totalExpected: totalExpected,
		startedAt:     time.Now(),
		status:        "running",
	}
}

// Listen drains progressCh until it is closed, accumulating counts.
// Call in a dedicated goroutine — blocks until the channel closes.
func (t *Tracker) Listen(progressCh <-chan models.ProgressEvent) {
	for event := range progressCh {
		t.processedCount.Add(int64(event.Processed))
		t.errorCount.Add(int64(event.Errors))
	}
}

// SetFinished records the finish time and final status string.
// Call once after the pipeline goroutine exits.
func (t *Tracker) SetFinished(status string) {
	now := time.Now()
	t.mu.Lock()
	t.finishedAt = &now
	t.status = status
	t.mu.Unlock()
}

// Snapshot returns a point-in-time copy of the current progress metrics.
// Safe to call concurrently from any goroutine.
func (t *Tracker) Snapshot() models.ProgressMetrics {
	t.mu.RLock()
	ft := t.finishedAt
	status := t.status
	t.mu.RUnlock()

	processed := t.processedCount.Load()
	elapsed := time.Since(t.startedAt).Seconds()

	pct := -1.0
	if t.totalExpected > 0 {
		pct = float64(processed) / float64(t.totalExpected) * 100
		if pct > 100 {
			pct = 100
		}
	}

	rate := 0.0
	if elapsed > 0 {
		rate = float64(processed) / elapsed
	}

	return models.ProgressMetrics{
		JobID:           t.jobID,
		Status:          status,
		ProcessedCount:  processed,
		ErrorCount:      t.errorCount.Load(),
		PercentComplete: pct,
		RecordsPerSec:   rate,
		StartTime:       t.startedAt,
		EndTime:         ft,
		ElapsedSeconds:  elapsed,
	}
}
