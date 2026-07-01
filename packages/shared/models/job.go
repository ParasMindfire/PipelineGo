package models

import (
	"fmt"
	"strings"
	"time"
)

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

// UnmarshalJSON decodes a status string label back into a JobStatus int,
// making the type symmetrically JSON-encodable for clients and tests.
func (s *JobStatus) UnmarshalJSON(b []byte) error {
	label := strings.Trim(string(b), `"`)
	for i, v := range [...]string{"pending", "running", "completed", "failed", "cancelled"} {
		if v == label {
			*s = JobStatus(i)
			return nil
		}
	}
	return fmt.Errorf("unknown job status %q", label)
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
