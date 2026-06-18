package ingestion

import (
	"context"
	"testing"

	"pipeline/internal/models"
)

func TestJSONReader(t *testing.T) {
	r := &JSONReader{URL: "https://jsonplaceholder.typicode.com/posts"}
	out := make(chan models.Record, 150)

	err := r.Read(context.Background(), out)
	close(out)

	if err != nil {
		t.Fatalf("json reader failed: %v", err)
	}

	count := 0
	for range out {
		count++
	}

	if count == 0 {
		t.Error("expected records from JSONPlaceholder, got 0")
	}
	t.Logf("received %d records", count)
}

func TestAPIReader(t *testing.T) {
	r := &APIReader{URL: "https://api.open-meteo.com/v1/forecast?latitude=20&longitude=85&current_weather=true"}
	out := make(chan models.Record, 5)

	err := r.Read(context.Background(), out)
	close(out)

	if err != nil {
		t.Fatalf("api reader failed: %v", err)
	}

	record := <-out
	if _, ok := record.Data["temperature"]; !ok {
		t.Error("expected 'temperature' field in Data")
	}
	t.Logf("weather data: %v", record.Data)
}
