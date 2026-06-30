package models

import "time"

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
