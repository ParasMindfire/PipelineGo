package ingestion

import (
	"context"
	"testing"

	"pipeline/internal/models"
)

func TestStartIngestion(t *testing.T) {
	sources := []models.SourceConfig{
		{Type: "csv", URL: "https://people.sc.fsu.edu/~jburkardt/data/csv/hw_200.csv"},
		{Type: "json", URL: "https://jsonplaceholder.typicode.com/posts"},
	}

	ch := StartIngestion(context.Background(), sources, 300)

	count := 0
	for range ch {
		count++
	}

	if count < 300 {
		t.Errorf("expected at least 300 records (200 csv + 100 json), got %d", count)
	}
	t.Logf("total records from all sources: %d", count)
}

func TestStartIngestion_UnknownType(t *testing.T) {
	sources := []models.SourceConfig{
		{Type: "xml", URL: "https://example.com/data.xml"},
	}

	ch := StartIngestion(context.Background(), sources, 10)

	count := 0
	for range ch {
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 records for unknown source type, got %d", count)
	}
}

func TestStartIngestion_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sources := []models.SourceConfig{
		{Type: "csv", URL: "https://people.sc.fsu.edu/~jburkardt/data/csv/hw_200.csv"},
		{Type: "json", URL: "https://jsonplaceholder.typicode.com/posts"},
	}

	ch := StartIngestion(ctx, sources, 300)

	read := 0
	for range ch {
		read++
		if read == 5 {
			cancel()
			break
		}
	}

	// drain remaining buffered records
	for range ch {
	}

	t.Logf("read %d records before cancellation", read)
}
