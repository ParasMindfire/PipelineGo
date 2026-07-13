package routes

import "time"

const (
	// Rate limiting applied globally to all HTTP routes.
	rateLimitRequests = 100
	rateLimitWindow   = time.Minute
)
