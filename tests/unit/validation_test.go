//go:build unit

package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"pipeline/apps/server/controller"
	"pipeline/packages/shared/models"
)

func TestValidateJobSpec_NoSources(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Export: models.ExportConfig{Type: "json", Path: "data/out.json"},
	})
	assert.EqualError(t, err, "at least one source is required")
}

func TestValidateJobSpec_InvalidSourceType(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "ftp", URL: "http://example.com/data.csv"}},
		Export:  models.ExportConfig{Type: "json", Path: "data/out.json"},
	})
	assert.ErrorContains(t, err, "invalid type")
}

func TestValidateJobSpec_InvalidSourceURL(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "not-a-url"}},
		Export:  models.ExportConfig{Type: "json", Path: "data/out.json"},
	})
	assert.ErrorContains(t, err, "url must be an absolute http or https URL")
}

func TestValidateJobSpec_InvalidExportType(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:  models.ExportConfig{Type: "xml", Path: "data/out.xml"},
	})
	assert.ErrorContains(t, err, "invalid type")
}

func TestValidateJobSpec_PathTraversal(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:  models.ExportConfig{Type: "json", Path: "../../etc/passwd"},
	})
	assert.ErrorContains(t, err, "must not contain '..'")
}

func TestValidateJobSpec_AbsolutePath(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:  models.ExportConfig{Type: "json", Path: "/etc/passwd"},
	})
	assert.ErrorContains(t, err, "must be relative")
}

func TestValidateJobSpec_NegativeWorkers(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources:     []models.SourceConfig{{Type: "csv", URL: "http://example.com/d.csv"}},
		Export:      models.ExportConfig{Type: "json", Path: "data/out.json"},
		Concurrency: models.ConcurrencyConfig{ValidationWorkers: -1},
	})
	assert.ErrorContains(t, err, "must not be negative")
}

func TestValidateJobSpec_Valid(t *testing.T) {
	err := controller.ValidateJobSpec(models.JobSpec{
		Sources: []models.SourceConfig{{Type: "api", URL: "https://api.example.com/data"}},
		Export:  models.ExportConfig{Type: "csv", Path: "data/output/result.csv"},
	})
	assert.NoError(t, err)
}

func TestValidateExportPath(t *testing.T) {
	cases := []struct {
		path    string
		wantErr bool
	}{
		{"data/output/job.json", false},
		{"out.json", false},
		{"", true},
		{"/etc/passwd", true},
		{"../../etc/passwd", true},
		{"a/../b", true},
	}
	for _, tc := range cases {
		err := controller.ValidateExportPath(tc.path)
		if tc.wantErr {
			assert.Error(t, err, "path=%q should fail", tc.path)
		} else {
			assert.NoError(t, err, "path=%q should pass", tc.path)
		}
	}
}

func TestValidateSourceURL(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com/data.csv", false},
		{"https://api.example.com/v1/data", false},
		{"not-a-url", true},
		{"ftp://example.com/file", true},
		{"", true},
		{"http://", true},
	}
	for _, tc := range cases {
		err := controller.ValidateSourceURL(tc.url)
		if tc.wantErr {
			assert.Error(t, err, "url=%q should fail", tc.url)
		} else {
			assert.NoError(t, err, "url=%q should pass", tc.url)
		}
	}
}
