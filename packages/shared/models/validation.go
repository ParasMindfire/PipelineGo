package models

import "time"

// ValidationError is one rule violation found in a Record.
type ValidationError struct {
	JobID    string    `json:"job_id"`
	RecordID string    `json:"record_id"`
	Field    string    `json:"field"`
	Message  string    `json:"message"`
	At       time.Time `json:"at"`
}
