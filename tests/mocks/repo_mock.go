package mocks

import (
	"sync"

	"pipeline/apps/server/repository"
	"pipeline/packages/shared/models"
)

// MockPipelineRepository is a thread-safe in-memory stub implementing
// service.PipelineRepo. Tests configure ErrToReturn or seed Jobs/Errors
// before calling service methods.
type MockPipelineRepository struct {
	mu          sync.Mutex
	Jobs        map[string]models.PipelineJob
	Errors      []models.ValidationError
	Aggregation map[string]models.AggregationResult
	ErrToReturn error // returned by every write operation when set
}

func NewMockRepo() *MockPipelineRepository {
	return &MockPipelineRepository{
		Jobs:        make(map[string]models.PipelineJob),
		Aggregation: make(map[string]models.AggregationResult),
	}
}

func (m *MockPipelineRepository) CreateJob(job models.PipelineJob) error {
	if m.ErrToReturn != nil {
		return m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Jobs[job.ID] = job
	return nil
}

func (m *MockPipelineRepository) GetJob(id string) (*models.PipelineJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.Jobs[id]
	if !ok {
		return nil, repository.ErrJobNotFound
	}
	return &job, nil
}

func (m *MockPipelineRepository) ListJobs() ([]models.PipelineJob, error) {
	if m.ErrToReturn != nil {
		return nil, m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	jobs := make([]models.PipelineJob, 0, len(m.Jobs))
	for _, j := range m.Jobs {
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (m *MockPipelineRepository) DeleteJob(id string) error {
	if m.ErrToReturn != nil {
		return m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Jobs, id)
	return nil
}

func (m *MockPipelineRepository) MarkStarted(id string) error { return m.ErrToReturn }
func (m *MockPipelineRepository) UpdateStatus(_ string, _ models.JobStatus) error {
	return m.ErrToReturn
}
func (m *MockPipelineRepository) MarkFinished(_ string, _ models.JobStatus, _, _ int) error {
	return m.ErrToReturn
}

func (m *MockPipelineRepository) SaveError(e models.ValidationError) error {
	if m.ErrToReturn != nil {
		return m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors = append(m.Errors, e)
	return nil
}

func (m *MockPipelineRepository) GetErrors(_ string) ([]models.ValidationError, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Errors, m.ErrToReturn
}

func (m *MockPipelineRepository) SaveAggregation(r models.AggregationResult) error {
	if m.ErrToReturn != nil {
		return m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Aggregation[r.JobID] = r
	return nil
}

func (m *MockPipelineRepository) GetAggregation(jobID string) (*models.AggregationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.Aggregation[jobID]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (m *MockPipelineRepository) CountsByStatus() (map[models.JobStatus]int, error) {
	if m.ErrToReturn != nil {
		return nil, m.ErrToReturn
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make(map[models.JobStatus]int)
	for _, j := range m.Jobs {
		counts[j.Status]++
	}
	return counts, nil
}
