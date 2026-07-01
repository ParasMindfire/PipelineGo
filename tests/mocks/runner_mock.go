package mocks

import (
	"context"

	"pipeline/packages/shared/models"
	"pipeline/packages/shared/pipelines/metrics"
)

// MockRunner is a no-op JobRunner used in unit/integration tests so pipeline
// goroutines are never actually spawned.
type MockRunner struct{}

func (m *MockRunner) Run(_ context.Context, _ models.JobSpec, _ string) {}

func (m *MockRunner) GetTracker(_ string) *metrics.Tracker { return nil }
