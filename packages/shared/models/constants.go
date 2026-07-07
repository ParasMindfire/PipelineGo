package models

const (
	// Default worker counts and buffer sizes used when a job omits concurrency config.
	DefaultValidationWorkers   = 5
	DefaultTransformWorkers    = 5
	DefaultIngestionBufferSize = 100
)
