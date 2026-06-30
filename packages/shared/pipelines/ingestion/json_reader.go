package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"pipeline/packages/shared/models"
)

// Read fetches the URL, decodes the JSON array, and sends one Record per element.
// Returns an error if the HTTP request fails, the status is not 200, or JSON is malformed.
func (r *JSONReader) Read(ctx context.Context, out chan<- models.Record) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("json reader: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("json reader: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("json reader: unexpected status %d from %s", resp.StatusCode, r.URL)
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return fmt.Errorf("json reader: decode: %w", err)
	}

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		now := time.Now()
		out <- models.Record{
			ID:          uuid.New().String(),
			Source:      r.URL,
			SourceType:  "json",
			Data:        item,
			IsValid:     false,
			ProcessedAt: &now,
		}
	}
	return nil
}
