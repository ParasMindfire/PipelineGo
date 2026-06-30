package controller

import "pipeline/apps/server/service"

// PipelineController holds the service the HTTP layer needs.
type PipelineController struct {
	service *service.PipelineService
}
