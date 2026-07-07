package ingestion

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"pipeline/packages/shared/models"
)

// Read fetches the CSV, parses headers from the first row, then sends one
// Record per data row into out. Stops early if ctx is cancelled.
func (r *CSVReader) Read(ctx context.Context, out chan<- models.Record) error {
	resp, err := fetchWithRetry(ctx, r.URL)
	if err != nil {
		return fmt.Errorf("csv reader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("csv reader: unexpected status %d from %s", resp.StatusCode, r.URL)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true // tolerate non-standard quoting in real-world CSVs

	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("csv reader: read headers: %w", err)
	}

	// strip BOM (\xef\xbb\xbf) and whitespace that Windows CSV exports add to the first header
	for i, h := range headers {
		headers[i] = strings.TrimSpace(strings.TrimPrefix(h, "\xef\xbb\xbf"))
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		row, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("csv reader: read row: %w", err)
		}

		data := make(map[string]interface{}, len(headers))
		for i, h := range headers {
			if i < len(row) {
				data[h] = strings.TrimSpace(row[i])
			}
		}

		now := time.Now()
		out <- models.Record{
			ID:          uuid.New().String(),
			Source:      r.URL,
			SourceType:  "csv",
			Data:        data,
			IsValid:     false,
			ProcessedAt: &now,
		}
	}
}
