//go:build unit

package unit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/transformation"
)

func TestTransform_StringToFloat(t *testing.T) {
	r := transformation.Transform(makeRecord("r1", map[string]interface{}{
		"price": "9.99",
	}))
	assert.Equal(t, 9.99, r.Data["price"])
}

func TestTransform_TrimsAndLowercases(t *testing.T) {
	r := transformation.Transform(makeRecord("r1", map[string]interface{}{
		"name": "  Alice Smith  ",
	}))
	assert.Equal(t, "alice smith", r.Data["name"])
}

func TestTransform_PassesThroughNonString(t *testing.T) {
	r := transformation.Transform(makeRecord("r1", map[string]interface{}{
		"score": 42.0,
	}))
	assert.Equal(t, 42.0, r.Data["score"])
}

func TestTransform_SetsProcessedAt(t *testing.T) {
	r := transformation.Transform(makeRecord("r1", nil))
	assert.NotNil(t, r.ProcessedAt)
}

func TestTransformation_WorkerPool_ClosesChannel(t *testing.T) {
	ctx := context.Background()
	in := feedChannel([]models.Record{
		makeRecord("r1", map[string]interface{}{"price": "5.0"}),
		makeRecord("r2", map[string]interface{}{"name": "BOB"}),
	})

	out := transformation.StartTransformation(ctx, in, 2)

	var count int
	for range out {
		count++
	}
	assert.Equal(t, 2, count)
}
