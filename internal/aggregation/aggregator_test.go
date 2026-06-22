package aggregation

import (
	"context"
	"fmt"
	"testing"

	"pipeline/internal/models"
)

func TestAggregation_Stats(t *testing.T) {
	in := make(chan models.Record, 10)

	for i := 0; i < 10; i++ {
		in <- models.Record{
			ID:         fmt.Sprintf("r%d", i),
			Source:     "test",
			SourceType: "csv",
			IsValid:    true,
			Data:       map[string]interface{}{"score": float64(i * 10)},
		}
	}
	close(in)

	exportCh, resultCh := StartAggregation(context.Background(), "job1", in)

	exported := 0
	for range exportCh {
		exported++
	}

	result := <-resultCh

	if result.TotalCount != 10 {
		t.Errorf("TotalCount: want 10, got %d", result.TotalCount)
	}
	if result.ValidCount != 10 {
		t.Errorf("ValidCount: want 10, got %d", result.ValidCount)
	}
	if exported != 10 {
		t.Errorf("exported records: want 10, got %d", exported)
	}

	stats := result.NumericStats["score"]
	if stats.Min != 0 {
		t.Errorf("Min: want 0, got %f", stats.Min)
	}
	if stats.Max != 90 {
		t.Errorf("Max: want 90, got %f", stats.Max)
	}
	if stats.Avg != 45 {
		t.Errorf("Avg: want 45, got %f", stats.Avg)
	}
	if stats.Sum != 450 {
		t.Errorf("Sum: want 450, got %f", stats.Sum)
	}
	if stats.Count != 10 {
		t.Errorf("Count: want 10, got %d", stats.Count)
	}

	t.Logf("score stats: min=%.0f max=%.0f avg=%.0f sum=%.0f count=%d",
		stats.Min, stats.Max, stats.Avg, stats.Sum, stats.Count)
}

func TestAggregation_BySource(t *testing.T) {
	in := make(chan models.Record, 6)
	in <- models.Record{ID: "1", Source: "a", SourceType: "csv", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "2", Source: "a", SourceType: "csv", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "3", Source: "b", SourceType: "json", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "4", Source: "c", SourceType: "api", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "5", Source: "c", SourceType: "api", IsValid: false, Data: map[string]interface{}{}}
	in <- models.Record{ID: "6", Source: "c", SourceType: "api", IsValid: true, Data: map[string]interface{}{}}
	close(in)

	exportCh, resultCh := StartAggregation(context.Background(), "job2", in)
	for range exportCh {
	}
	result := <-resultCh

	if result.BySource["csv"] != 2 {
		t.Errorf("csv count: want 2, got %d", result.BySource["csv"])
	}
	if result.BySource["json"] != 1 {
		t.Errorf("json count: want 1, got %d", result.BySource["json"])
	}
	if result.BySource["api"] != 3 {
		t.Errorf("api count: want 3, got %d", result.BySource["api"])
	}
	if result.ValidCount != 5 {
		t.Errorf("ValidCount: want 5, got %d", result.ValidCount)
	}
	t.Logf("by_source: %v", result.BySource)
}

func TestAggregation_PassThrough(t *testing.T) {
	in := make(chan models.Record, 3)
	in <- models.Record{ID: "1", Source: "test", SourceType: "csv", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "2", Source: "test", SourceType: "csv", IsValid: true, Data: map[string]interface{}{}}
	in <- models.Record{ID: "3", Source: "test", SourceType: "csv", IsValid: true, Data: map[string]interface{}{}}
	close(in)

	exportCh, resultCh := StartAggregation(context.Background(), "job3", in)

	exported := 0
	for range exportCh {
		exported++
	}
	<-resultCh

	if exported != 3 {
		t.Errorf("pass-through: want 3 records in exportCh, got %d", exported)
	}
}
