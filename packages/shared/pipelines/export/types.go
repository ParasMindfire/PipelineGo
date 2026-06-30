package export

import "pipeline/packages/shared/models"

// AggregationSaver is the only store method the exporter needs.
// Using an interface here lets tests inject a stub without a real DB.
type AggregationSaver interface {
	SaveAggregation(models.AggregationResult) error
}

// Exporter writes pipeline output records to a JSON file and persists the
// aggregation result to PostgreSQL once all records have been written.
type Exporter struct{ store AggregationSaver }
