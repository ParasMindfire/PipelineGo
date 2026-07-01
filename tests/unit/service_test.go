//go:build unit

package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pipeline/apps/server/service"
	"pipeline/packages/shared/models"
	"pipeline/tests/mocks"
)

func newTestService() (*service.PipelineService, *mocks.MockPipelineRepository) {
	repo := mocks.NewMockRepo()
	runner := &mocks.MockRunner{}
	svc := service.NewPipelineService(repo, runner)
	return svc, repo
}

func TestCreatePipeline_NoSources(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.CreatePipeline(models.JobSpec{})
	assert.ErrorIs(t, err, service.ErrNoSources)
}

func TestCreatePipeline_PersistsJob(t *testing.T) {
	svc, repo := newTestService()
	spec := models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:  models.ExportConfig{Type: "json", Path: "data/out.json"},
	}
	job, err := svc.CreatePipeline(spec)
	require.NoError(t, err)
	assert.NotEmpty(t, job.ID)
	assert.Equal(t, models.StatusPending, job.Status)

	_, ok := repo.Jobs[job.ID]
	assert.True(t, ok, "job should be persisted in repo")
}

func TestCreatePipeline_AppliesDefaultConcurrency(t *testing.T) {
	svc, _ := newTestService()
	spec := models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:  models.ExportConfig{Type: "json", Path: "data/out.json"},
	}
	job, err := svc.CreatePipeline(spec)
	require.NoError(t, err)
	assert.Equal(t, 5, job.Spec.Concurrency.ValidationWorkers)
	assert.Equal(t, 5, job.Spec.Concurrency.TransformWorkers)
}

func TestGetPipeline_NotFound(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.GetPipeline("non-existent-id")
	assert.ErrorIs(t, err, service.ErrJobNotFound)
}

func TestListPipelines_Empty(t *testing.T) {
	svc, _ := newTestService()
	jobs, err := svc.ListPipelines()
	require.NoError(t, err)
	assert.Empty(t, jobs)
}

func TestListPipelines_ReturnsAll(t *testing.T) {
	svc, repo := newTestService()
	repo.Jobs["a"] = models.PipelineJob{ID: "a"}
	repo.Jobs["b"] = models.PipelineJob{ID: "b"}

	jobs, err := svc.ListPipelines()
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestCancelPipeline_NotRunning(t *testing.T) {
	svc, _ := newTestService()
	err := svc.CancelPipeline("some-id")
	assert.ErrorIs(t, err, service.ErrJobNotRunning)
}

func TestJobCounts_Empty(t *testing.T) {
	svc, _ := newTestService()
	counts, err := svc.JobCounts()
	require.NoError(t, err)
	assert.Equal(t, 0, counts.Total)
}

func TestJobCounts_AggregatesCorrectly(t *testing.T) {
	svc, repo := newTestService()
	repo.Jobs["1"] = models.PipelineJob{ID: "1", Status: models.StatusCompleted}
	repo.Jobs["2"] = models.PipelineJob{ID: "2", Status: models.StatusCompleted}
	repo.Jobs["3"] = models.PipelineJob{ID: "3", Status: models.StatusRunning}
	repo.Jobs["4"] = models.PipelineJob{ID: "4", Status: models.StatusFailed}

	counts, err := svc.JobCounts()
	require.NoError(t, err)
	assert.Equal(t, 4, counts.Total)
	assert.Equal(t, 2, counts.Completed)
	assert.Equal(t, 1, counts.Running)
	assert.Equal(t, 1, counts.Failed)
}
