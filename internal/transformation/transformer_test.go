package transformation

import (
	"context"
	"strings"
	"testing"

	"pipeline/internal/models"
)

func TestTransform(t *testing.T) {
	r := models.Record{
		ID:     "1",
		Source: "test",
		Data: map[string]interface{}{
			"new_cases":   "50000",
			"location":    "  INDIA  ",
			"temperature": 28.5,
			"is_active":   true,
		},
	}

	result := Transform(r)

	if result.ProcessedAt == nil {
		t.Error("ProcessedAt should be set after transform")
	}

	if v, ok := result.Data["new_cases"].(float64); !ok || v != 50000 {
		t.Errorf("new_cases: expected float64(50000), got %v (%T)", result.Data["new_cases"], result.Data["new_cases"])
	}

	if result.Data["location"] != "india" {
		t.Errorf("location: expected 'india', got %v", result.Data["location"])
	}

	if result.Data["temperature"] != 28.5 {
		t.Errorf("temperature: expected 28.5, got %v", result.Data["temperature"])
	}

	if result.Data["is_active"] != true {
		t.Errorf("is_active: expected true, got %v", result.Data["is_active"])
	}
}

func TestTransform_EmptyString(t *testing.T) {
	r := models.Record{
		ID:     "1",
		Source: "test",
		Data: map[string]interface{}{
			"location": "   ",
		},
	}

	result := Transform(r)
	if result.Data["location"] != "" {
		t.Errorf("expected empty string after trimming whitespace, got %q", result.Data["location"])
	}
}

func TestStartTransformation(t *testing.T) {
	in := make(chan models.Record, 5)
	in <- models.Record{ID: "1", Source: "test", IsValid: true, Data: map[string]interface{}{"score": "99"}}
	in <- models.Record{ID: "2", Source: "test", IsValid: true, Data: map[string]interface{}{"city": "  LONDON  "}}
	in <- models.Record{ID: "3", Source: "test", IsValid: true, Data: map[string]interface{}{"score": "42"}}
	close(in)

	out := StartTransformation(context.Background(), in, 2)

	var results []models.Record
	for r := range out {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 records, got %d", len(results))
	}

	for _, r := range results {
		for _, val := range r.Data {
			if v, ok := val.(string); ok {
				if v != strings.ToLower(v) {
					t.Errorf("string not lowercased: %q", v)
				}
			}
		}
	}
	t.Logf("transformed %d records", len(results))
}
