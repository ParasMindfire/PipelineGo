package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"pipeline/internal/models"
)

// APIReader fetches a single-object JSON response and flattens it into one Record.
// Designed for REST APIs like Open-Meteo that return one object, not an array.
type APIReader struct{ URL string }

// Read fetches the URL, decodes the JSON object, flattens any nested "current_weather"
// block into Record.Data, and sends a single Record into out.
func (r *APIReader) Read(ctx context.Context, out chan<- models.Record) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("api reader: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("api reader: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api reader: unexpected status %d from %s", resp.StatusCode, r.URL)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("api reader: decode: %w", err)
	}

	// Open-Meteo returns { current_weather: { temperature, windspeed, ... }, latitude, longitude }
	// Flatten the nested block so all fields sit at the top level of Record.Data.
	data := make(map[string]interface{})
	if cw, ok := raw["current_weather"].(map[string]interface{}); ok {
		for k, v := range cw {
			data[k] = v
		}
	}
	data["latitude"] = raw["latitude"]
	data["longitude"] = raw["longitude"]

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	now := time.Now()
	out <- models.Record{
		ID:          uuid.New().String(),
		Source:      r.URL,
		SourceType:  "api",
		Data:        data,
		IsValid:     false,
		ProcessedAt: &now,
	}
	return nil
}
