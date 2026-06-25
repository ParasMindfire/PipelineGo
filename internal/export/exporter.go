package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"pipeline/internal/models"
)

// AggregationSaver is the only store method the exporter needs.
// Using an interface here lets tests inject a stub without a real DB.
type AggregationSaver interface {
	SaveAggregation(models.AggregationResult) error
}

// Exporter writes pipeline output records to a JSON file and persists the
// aggregation result to PostgreSQL once all records have been written.
type Exporter struct{ store AggregationSaver }

// NewExporter creates an Exporter backed by the given store.
func NewExporter(s AggregationSaver) *Exporter { return &Exporter{store: s} }

// Run reads all records from exportCh and writes them as a JSON array to the
// file at cfg.Path. After exportCh closes it reads one AggregationResult from
// aggCh and saves it to the store.
func (e *Exporter) Run(
	ctx context.Context,
	jobID string,
	exportCh <-chan models.Record,
	aggCh <-chan models.AggregationResult,
	cfg models.ExportConfig,
) error {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(cfg.Path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)

	if _, err := f.WriteString("[\n"); err != nil {
		return fmt.Errorf("write opening bracket: %w", err)
	}

	first := true
	for record := range exportCh {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !first {
			if _, err := f.WriteString(",\n"); err != nil {
				return fmt.Errorf("write separator: %w", err)
			}
		}
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("encode record %s: %w", record.ID, err)
		}
		first = false
	}

	if _, err := f.WriteString("\n]"); err != nil {
		return fmt.Errorf("write closing bracket: %w", err)
	}

	if agg, ok := <-aggCh; ok {
		if err := e.store.SaveAggregation(agg); err != nil {
			return fmt.Errorf("save aggregation: %w", err)
		}
	}

	return nil
}
