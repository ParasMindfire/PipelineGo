package ingestion

import "time"

const (
	maxRetries     = 3
	retryBaseDelay = time.Second
	httpTimeout    = 15 * time.Second
)
