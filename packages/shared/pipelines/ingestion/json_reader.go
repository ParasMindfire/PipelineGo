package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pipeline/packages/shared/models"
)

// Read fetches the URL, decodes the JSON array, and sends one Record per element.
// Returns an error if the HTTP request fails, the status is not 200, or JSON is malformed.
func (r *JSONReader) Read(ctx context.Context, out chan<- models.Record) error {
	resp, err := fetchWithRetry(ctx, r.URL)
	if err != nil {
		return fmt.Errorf("json reader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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
