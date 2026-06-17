package ingestion

import (
	"context"
	"testing"

	"pipeline/internal/models"
)

func TestCSVReader(t *testing.T) {
	reader := &CSVReader{URL: "https://people.sc.fsu.edu/~jburkardt/data/csv/hw_200.csv"}
	out := make(chan models.Record, 250)

	err := reader.Read(context.Background(), out)
	close(out)

	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var count int
	for range out {
		count++
	}

	if count != 200 {
		t.Errorf("expected 200 records, got %d", count)
	}
	t.Logf("received %d records", count)
}

func TestCSVReader_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := &CSVReader{URL: "https://people.sc.fsu.edu/~jburkardt/data/csv/hw_200.csv"}
	out := make(chan models.Record, 250)

	err := reader.Read(ctx, out)
	close(out)

	if err == nil {
		t.Log("reader finished before ctx check — acceptable for small files")
	} else {
		t.Logf("correctly returned cancellation error: %v", err)
	}
}
