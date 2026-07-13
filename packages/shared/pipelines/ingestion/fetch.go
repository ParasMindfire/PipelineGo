package ingestion

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: httpTimeout}

// fetchWithRetry performs a GET to url with up to maxRetries attempts.
// Retries on network errors and 5xx responses using exponential backoff
// (1s, 2s, 4s, ...). Returns immediately on 4xx — those won't improve on retry.
// Respects ctx cancellation during backoff waits.
func fetchWithRetry(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
