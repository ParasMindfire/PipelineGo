package mocks

import (
	"pipeline/apps/server/service"
	"pipeline/packages/shared/models"
)

// MockPipelineService is a configurable stub implementing controller.PipelineService.
// Tests set Jobs/Errors/ErrToReturn before calling controller handlers.
type MockPipelineService struct {
	Jobs        map[string]models.PipelineJob
	Errors      []models.ValidationError
	ErrToReturn error
}

func NewMockService() *MockPipelineService {
	return &MockPipelineService{
		Jobs: make(map[string]models.PipelineJob),
	}
}

func (m *MockPipelineService) CreatePipeline(spec models.JobSpec) (models.PipelineJob, error) {
	if m.ErrToReturn != nil {
		return models.PipelineJob{}, m.ErrToReturn
	}
	job := models.PipelineJob{ID: "mock-job-id", Status: models.StatusPending, Spec: spec}
	m.Jobs[job.ID] = job
	return job, nil
}

func (m *MockPipelineService) ListPipelines() ([]models.PipelineJob, error) {
	if m.ErrToReturn != nil {
		return nil, m.ErrToReturn
	}
	jobs := make([]models.PipelineJob, 0, len(m.Jobs))
	for _, j := range m.Jobs {
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (m *MockPipelineService) GetPipeline(id string) (*models.PipelineJob, error) {
	if m.ErrToReturn != nil {
		return nil, m.ErrToReturn
	}
	j, ok := m.Jobs[id]
	if !ok {
		return nil, service.ErrJobNotFound
	}
	return &j, nil
}

func (m *MockPipelineService) GetProgress(_ string) (models.ProgressMetrics, error) {
	return models.ProgressMetrics{}, m.ErrToReturn
}

func (m *MockPipelineService) GetResults(_ string) (*models.AggregationResult, error) {
	return nil, m.ErrToReturn
}

func (m *MockPipelineService) GetErrors(_ string) ([]models.ValidationError, error) {
	return m.Errors, m.ErrToReturn
}

func (m *MockPipelineService) CancelPipeline(_ string) error { return m.ErrToReturn }

func (m *MockPipelineService) DeletePipeline(_ string) error { return m.ErrToReturn }

func (m *MockPipelineService) JobCounts() (service.JobCounts, error) {
	var c service.JobCounts
	for _, j := range m.Jobs {
		c.Total++
		switch j.Status {
		case models.StatusRunning:
			c.Running++
		case models.StatusCompleted:
			c.Completed++
		case models.StatusFailed, models.StatusCancelled:
			c.Failed++
		}
	}
	return c, m.ErrToReturn
}
