//go:build unit

package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/aggregation"
)

func TestAggregation_CountsTotalAndValid(t *testing.T) {
	ctx := context.Background()
	rec1 := makeRecord("r1", map[string]interface{}{"price": 10.0})
	rec1.IsValid = true
	rec2 := makeRecord("r2", map[string]interface{}{"price": 20.0})
	rec2.IsValid = false

	in := feedChannel([]models.Record{rec1, rec2})
	exportCh, aggCh := aggregation.StartAggregation(ctx, "job1", in)

	for range exportCh {
	}

	result, ok := <-aggCh
	require.True(t, ok)
	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, 1, result.ValidCount)
}

func TestAggregation_ComputesNumericStats(t *testing.T) {
	ctx := context.Background()
	in := feedChannel([]models.Record{
		makeRecord("r1", map[string]interface{}{"price": 10.0}),
		makeRecord("r2", map[string]interface{}{"price": 20.0}),
		makeRecord("r3", map[string]interface{}{"price": 30.0}),
	})
	exportCh, aggCh := aggregation.StartAggregation(ctx, "job1", in)
	for range exportCh {
	}

	result := <-aggCh
	stats, ok := result.NumericStats["price"]
	require.True(t, ok, "price stats should be computed")
	assert.Equal(t, 10.0, stats.Min)
	assert.Equal(t, 30.0, stats.Max)
	assert.Equal(t, 60.0, stats.Sum)
	assert.InDelta(t, 20.0, stats.Avg, 0.001)
	assert.Equal(t, 3, stats.Count)
}

func TestAggregation_GroupsBySource(t *testing.T) {
	ctx := context.Background()
	csvRec := makeRecord("r1", nil)
	csvRec.SourceType = "csv"
	apiRec := makeRecord("r2", nil)
	apiRec.SourceType = "api"

	in := feedChannel([]models.Record{csvRec, csvRec, apiRec})
	exportCh, aggCh := aggregation.StartAggregation(ctx, "job1", in)
	for range exportCh {
	}

	result := <-aggCh
	assert.Equal(t, 2, result.BySource["csv"])
	assert.Equal(t, 1, result.BySource["api"])
}

func TestAggregation_ClosesChannelsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	records := make([]models.Record, 50)
	for i := range records {
		records[i] = makeRecord("r", map[string]interface{}{"price": float64(i)})
	}
	in := feedChannel(records)

	exportCh, aggCh := aggregation.StartAggregation(ctx, "job1", in)
	cancel()

	done := make(chan struct{})
	go func() {
		for range exportCh {
		}
		for range aggCh {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("aggregation channels did not close after context cancellation")
	}
}
