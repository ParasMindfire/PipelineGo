//go:build unit

package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pipeline/apps/server/controller"
	"pipeline/packages/shared/models"
	"pipeline/tests/mocks"
)

func newTestController() (*controller.PipelineController, *mocks.MockPipelineService) {
	svc := mocks.NewMockService()
	ctrl := controller.NewPipelineController(svc)
	return ctrl, svc
}

func TestHealth_ReturnsOK(t *testing.T) {
	ctrl, _ := newTestController()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	ctrl.Health(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListPipelines_ReturnsJSON(t *testing.T) {
	ctrl, svc := newTestController()
	svc.Jobs["abc"] = models.PipelineJob{ID: "abc", Status: models.StatusCompleted}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
	w := httptest.NewRecorder()
	ctrl.ListPipelines(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var jobs []models.PipelineJob
	require.NoError(t, json.NewDecoder(w.Body).Decode(&jobs))
	assert.Len(t, jobs, 1)
}

func TestCreatePipeline_InvalidJSON_Returns400(t *testing.T) {
	ctrl, _ := newTestController()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	ctrl.CreatePipeline(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePipeline_FailsValidation_Returns400(t *testing.T) {
	ctrl, _ := newTestController()
	body := `{"sources":[{"type":"ftp","url":"bad-url"}],"export":{"type":"json","path":"out.json"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pipelines", strings.NewReader(body))
	w := httptest.NewRecorder()
	ctrl.CreatePipeline(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
