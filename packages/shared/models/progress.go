package models

import "time"

// ProgressMetrics is the shape returned by GET /api/v1/pipelines/{id}/progress.
type ProgressMetrics struct {
	JobID           string     `json:"job_id"`
	Status          string     `json:"status"`
	ProcessedCount  int64      `json:"processed_count"`
	ErrorCount      int64      `json:"error_count"`
	PercentComplete float64    `json:"percent_complete"` // 0–100, or -1 if total is unknown
	RecordsPerSec   float64    `json:"records_per_sec"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	ElapsedSeconds  float64    `json:"elapsed_seconds"`
}

// ProgressEvent is sent by pipeline stages to the progress tracker goroutine.
type ProgressEvent struct {
	Processed int
	Errors    int
}
