package controller

import (
	"pipeline/apps/server/service"
	"pipeline/packages/shared/models"
)

// PipelineService is the business-logic surface the controller depends on.
// Defined as an interface here so the controller can be tested with a mock
// service without wiring up a real repository or pipeline runner.
type PipelineService interface {
	CreatePipeline(spec models.JobSpec) (models.PipelineJob, error)
	ListPipelines() ([]models.PipelineJob, error)
	GetPipeline(id string) (*models.PipelineJob, error)
	GetProgress(id string) (models.ProgressMetrics, error)
	GetResults(id string) (*models.AggregationResult, error)
	GetErrors(id string) ([]models.ValidationError, error)
	CancelPipeline(id string) error
	DeletePipeline(id string) error
	JobCounts() (service.JobCounts, error)
}

// PipelineController holds the service the HTTP layer needs.
type PipelineController struct {
	service PipelineService
}
