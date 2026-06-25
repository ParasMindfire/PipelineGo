package export

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"pipeline/internal/models"
)

// stubStore satisfies AggregationSaver without a real DB.
type stubStore struct{ saved *models.AggregationResult }

func (s *stubStore) SaveAggregation(r models.AggregationResult) error {
	s.saved = &r
	return nil
}

func TestExporter_WritesFile(t *testing.T) {
	exportCh := make(chan models.Record, 5)
	aggCh := make(chan models.AggregationResult, 1)

	now := time.Now()
	exportCh <- models.Record{
		ID: "r1", Source: "test", SourceType: "csv", IsValid: true,
		ProcessedAt: &now, Data: map[string]interface{}{"score": 42.0},
	}
	exportCh <- models.Record{
		ID: "r2", Source: "test", SourceType: "csv", IsValid: true,
		ProcessedAt: &now, Data: map[string]interface{}{"score": 88.0},
	}
	close(exportCh)

	aggCh <- models.AggregationResult{
		JobID: "job-test", TotalCount: 2, ValidCount: 2,
		ComputedAt: time.Now(),
	}
	close(aggCh)

	path := "../../data/output/exporter_test.json"
	if err := os.MkdirAll("../../data/output", 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	stub := &stubStore{}
	exp := NewExporter(stub)

	if err := exp.Run(context.Background(), "job-test", exportCh, aggCh, models.ExportConfig{Path: path}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
	t.Logf("output file: %s (%d bytes)", path, info.Size())

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var records []models.Record
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("output file is not valid JSON: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records in file, got %d", len(records))
	}
	t.Logf("file contains %d valid JSON records", len(records))

	if stub.saved == nil {
		t.Error("SaveAggregation was not called")
	} else {
		t.Logf("aggregation saved: total=%d", stub.saved.TotalCount)
	}
}
