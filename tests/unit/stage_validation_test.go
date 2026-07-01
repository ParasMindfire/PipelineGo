//go:build unit

package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/validation"
)

func TestValidation_ValidRecord(t *testing.T) {
	ok, errs := validation.Validate("job1", makeRecord("rec1", map[string]interface{}{
		"price": "12.50",
	}))
	assert.True(t, ok)
	assert.Empty(t, errs)
}

func TestValidation_MissingID(t *testing.T) {
	ok, errs := validation.Validate("job1", models.Record{Source: "http://x.com"})
	assert.False(t, ok)
	assert.Len(t, errs, 1)
	assert.Equal(t, "id", errs[0].Field)
}

func TestValidation_NonNumericField(t *testing.T) {
	ok, errs := validation.Validate("job1", makeRecord("r1", map[string]interface{}{
		"price": "not-a-number",
	}))
	assert.False(t, ok)
	assert.Len(t, errs, 1)
	assert.Equal(t, "price", errs[0].Field)
}

func TestValidation_IgnoresUnknownFields(t *testing.T) {
	ok, errs := validation.Validate("job1", makeRecord("r1", map[string]interface{}{
		"name": "Alice",
	}))
	assert.True(t, ok)
	assert.Empty(t, errs)
}

func TestValidation_WorkerPool_ClosesChannels(t *testing.T) {
	ctx := context.Background()
	in := feedChannel([]models.Record{
		makeRecord("r1", map[string]interface{}{"price": "10.0"}),
		makeRecord("r2", map[string]interface{}{"price": "bad"}),
	})

	validCh, errCh := validation.StartValidation(ctx, "job1", in, 2, nil)

	var valid []models.Record
	for r := range validCh {
		valid = append(valid, r)
	}
	var errs []models.ValidationError
	for e := range errCh {
		errs = append(errs, e)
	}

	assert.Len(t, valid, 1)
	assert.Len(t, errs, 1)
}

func TestValidation_WorkerPool_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	in := feedChannel([]models.Record{makeRecord("r1", nil)})
	validCh, errCh := validation.StartValidation(ctx, "job1", in, 2, nil)

	done := make(chan struct{})
	go func() {
		for range validCh {
		}
		for range errCh {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channels did not close after context cancellation")
	}
}
