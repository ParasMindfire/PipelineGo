package models

import "time"

// JobStatus — lifecycle of a pipeline job.
type JobStatus int

const (
	StatusPending   JobStatus = iota // 0
	StatusRunning                    // 1
	StatusCompleted                  // 2
	StatusFailed                     // 3
	StatusCancelled                  // 4
)

func (s JobStatus) String() string {
	return [...]string{"pending", "running", "completed", "failed", "cancelled"}[s]
}

// MarshalJSON encodes the status as its string label in API responses.
func (s JobStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// Record is one unit of data travelling through the pipeline.
// Data uses map[string]interface{} because CSV, JSON, and API sources all have
// different field shapes that are only known at runtime.
type Record struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	SourceType  string                 `json:"source_type"` // "csv" | "json" | "api"
	Data        map[string]interface{} `json:"data"`
	IsValid     bool                   `json:"is_valid"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
}

// ValidationError is one rule violation found in a Record.
type ValidationError struct {
	JobID    string    `json:"job_id"`
	RecordID string    `json:"record_id"`
	Field    string    `json:"field"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}

// ConcurrencyConfig holds worker counts per stage, configurable per job via the POST body.
type ConcurrencyConfig struct {
	ValidationWorkers   int `json:"validation_workers"`    // default 5
	TransformWorkers    int `json:"transform_workers"`     // default 5
	IngestionBufferSize int `json:"ingestion_buffer_size"` // channel buffer, default 100
}

// DefaultConcurrency returns sane defaults used when a job omits concurrency config.
func DefaultConcurrency() ConcurrencyConfig {
	return ConcurrencyConfig{
		ValidationWorkers:   5,
		TransformWorkers:    5,
		IngestionBufferSize: 100,
	}
}

// SourceConfig describes one input source.
type SourceConfig struct {
	Type string `json:"type"` // "csv" | "json" | "api"
	URL  string `json:"url"`
}

// ExportConfig describes where to write output.
type ExportConfig struct {
	Type string `json:"type"` // "json" | "csv"
	Path string `json:"path"` // e.g. "data/output/job-123.json"
}

// JobSpec is the JSON body of POST /api/v1/pipelines.
type JobSpec struct {
	Sources     []SourceConfig    `json:"sources"`
	Export      ExportConfig      `json:"export"`
	Concurrency ConcurrencyConfig `json:"concurrency"`
}

// PipelineJob tracks one running or completed pipeline job in the database.
type PipelineJob struct {
	ID          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	Spec        JobSpec    `json:"spec"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	ErrorCount  int        `json:"error_count"`
	RecordCount int        `json:"record_count"`
}

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

// FieldStats holds min/max/sum/avg for one numeric field across all records.
type FieldStats struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Count int     `json:"count"`
}

// AggregationResult holds the final computed statistics for a completed job.
type AggregationResult struct {
	JobID        string                `json:"job_id"`
	TotalCount   int                   `json:"total_count"`
	ValidCount   int                   `json:"valid_count"`
	ErrorCount   int                   `json:"error_count"`
	BySource     map[string]int        `json:"by_source"`
	NumericStats map[string]FieldStats `json:"numeric_stats"`
	ComputedAt   time.Time             `json:"computed_at"`
}

// ProgressEvent is sent by pipeline stages to the progress tracker goroutine.
type ProgressEvent struct {
	Processed int
	Errors    int
}
