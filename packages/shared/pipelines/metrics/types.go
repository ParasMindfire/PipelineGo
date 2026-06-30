package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Tracker accumulates progress events from pipeline workers and exposes
// a thread-safe Snapshot for the progress API endpoint.
type Tracker struct {
	jobID          string
	totalExpected  int64 // 0 = unknown → PercentComplete returns -1
	processedCount atomic.Int64
	errorCount     atomic.Int64
	startedAt      time.Time
	finishedAt     *time.Time
	status         string
	mu             sync.RWMutex
}
