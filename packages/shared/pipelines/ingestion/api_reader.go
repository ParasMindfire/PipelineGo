package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pipeline/packages/shared/models"
)

// Read fetches the URL, decodes the JSON object, flattens any nested "current_weather"
// block into Record.Data, and sends a single Record into out.
func (r *APIReader) Read(ctx context.Context, out chan<- models.Record) error {
	resp, err := fetchWithRetry(ctx, r.URL)
	if err != nil {
		return fmt.Errorf("api reader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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
