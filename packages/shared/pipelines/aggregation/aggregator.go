package aggregation

import (
	"context"
	"time"

	"pipeline/packages/shared/models"
)

// StartAggregation reads every transformed record, accumulates stats,
// passes each record through to exportOut, then emits one AggregationResult on aggOut.
//
// Only ONE goroutine reads from in — this is the fan-in point.
// The exporter reads exportOut, NOT the upstream channel directly.
func StartAggregation(
	ctx context.Context,
	jobID string,
	in <-chan models.Record,
) (exportOut <-chan models.Record, aggOut <-chan models.AggregationResult) {
	exportCh := make(chan models.Record, 100)
	resultCh := make(chan models.AggregationResult, 1)

	go func() {
		defer close(exportCh)
		defer close(resultCh)

		result := models.AggregationResult{
			JobID:        jobID,
			BySource:     make(map[string]int),
			NumericStats: make(map[string]models.FieldStats),
		}
		fieldCounts := make(map[string]int)

		for record := range in {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result.TotalCount++
			result.BySource[record.SourceType]++
			if record.IsValid {
				result.ValidCount++
			}

			for field, val := range record.Data {
				f, ok := val.(float64)
				if !ok {
					continue
				}

				stats := result.NumericStats[field]
				fieldCounts[field]++
				stats.Sum += f
				stats.Count = fieldCounts[field]

				if fieldCounts[field] == 1 {
					stats.Min = f
					stats.Max = f
				} else {
					if f < stats.Min {
						stats.Min = f
					}
					if f > stats.Max {
						stats.Max = f
					}
				}

				result.NumericStats[field] = stats
			}

			exportCh <- record
		}

		for field, stats := range result.NumericStats {
			if stats.Count > 0 {
				stats.Avg = stats.Sum / float64(stats.Count)
			}
			result.NumericStats[field] = stats
		}

		result.ComputedAt = time.Now()
		resultCh <- result
	}()

	return exportCh, resultCh
}
