package models

import "time"

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
