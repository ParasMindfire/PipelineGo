//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "pipeline/apps/server/docs"

	"pipeline/apps/server/controller"
	"pipeline/apps/server/routes"
	"pipeline/apps/server/service"
	"pipeline/packages/shared/models"
	"pipeline/tests/mocks"
)

const testAPIKey = "test-key-abc123"

func newTestServer(t *testing.T) (*httptest.Server, *mocks.MockPipelineRepository) {
	t.Helper()
	repo := mocks.NewMockRepo()
	runner := &mocks.MockRunner{}
	svc := service.NewPipelineService(repo, runner)
	ctrl := controller.NewPipelineController(svc)
	router := routes.NewRouter(ctrl, testAPIKey)
	return httptest.NewServer(router), repo
}

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListPipelines_Open(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/pipelines")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreatePipeline_NoAPIKey_Returns401(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	body := `{"sources":[{"type":"csv","url":"http://example.com/d.csv"}],"export":{"type":"json","path":"out.json"}}`
	resp, err := http.Post(srv.URL+"/api/v1/pipelines", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCreatePipeline_WithAPIKey_Returns201(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	body := `{"sources":[{"type":"csv","url":"http://example.com/d.csv"}],"export":{"type":"json","path":"out.json"}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var job models.PipelineJob
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&job))
	assert.NotEmpty(t, job.ID)
}

func TestGetPipeline_InvalidUUID_Returns400(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/pipelines/not-a-uuid")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetPipeline_NotFound_Returns404(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/pipelines/00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreatePipeline_InvalidBody_Returns400(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	body := `{"sources":[{"type":"ftp","url":"not-a-url"}],"export":{"type":"xml","path":"../../etc/passwd"}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/pipelines", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testAPIKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestVersionGate_UnsupportedVersion_Returns404(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v99/pipelines")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeletePipeline_NoAPIKey_Returns401(t *testing.T) {
	srv, repo := newTestServer(t)
	defer srv.Close()

	repo.Jobs["11111111-1111-1111-1111-111111111111"] = models.PipelineJob{
		ID: "11111111-1111-1111-1111-111111111111",
	}

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/pipelines/11111111-1111-1111-1111-111111111111", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
