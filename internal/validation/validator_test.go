package validation

import (
	"context"
	"testing"

	"pipeline/internal/models"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		record   models.Record
		valid    bool
		errCount int
	}{
		{
			name:     "valid record",
			record:   models.Record{ID: "1", Source: "http://x.com", Data: map[string]interface{}{}},
			valid:    true,
			errCount: 0,
		},
		{
			name:     "missing id",
			record:   models.Record{ID: "", Source: "http://x.com", Data: map[string]interface{}{}},
			valid:    false,
			errCount: 1,
		},
		{
			name:     "missing source",
			record:   models.Record{ID: "1", Source: "", Data: map[string]interface{}{}},
			valid:    false,
			errCount: 1,
		},
		{
			name:     "both missing",
			record:   models.Record{ID: "", Source: "", Data: map[string]interface{}{}},
			valid:    false,
			errCount: 2,
		},
		{
			name: "bad numeric field",
			record: models.Record{
				ID: "1", Source: "http://x.com",
				Data: map[string]interface{}{"new_cases": "abc"},
			},
			valid:    false,
			errCount: 1,
		},
		{
			name: "valid numeric field",
			record: models.Record{
				ID: "1", Source: "http://x.com",
				Data: map[string]interface{}{"new_cases": "12345"},
			},
			valid:    true,
			errCount: 0,
		},
		{
			name: "already a float — skipped by numeric check",
			record: models.Record{
				ID: "1", Source: "http://x.com",
				Data: map[string]interface{}{"temperature": 28.5},
			},
			valid:    true,
			errCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, errs := Validate("job1", tc.record)
			if ok != tc.valid {
				t.Errorf("valid: got %v want %v", ok, tc.valid)
			}
			if len(errs) != tc.errCount {
				t.Errorf("error count: got %d want %d — errors: %v", len(errs), tc.errCount, errs)
			}
		})
	}
}

func TestStartValidation(t *testing.T) {
	in := make(chan models.Record, 10)

	in <- models.Record{ID: "1", Source: "http://x.com", Data: map[string]interface{}{}}
	in <- models.Record{ID: "2", Source: "http://x.com", Data: map[string]interface{}{}}
	in <- models.Record{ID: "3", Source: "http://x.com", Data: map[string]interface{}{}}
	in <- models.Record{ID: "", Source: "http://x.com", Data: map[string]interface{}{}}
	in <- models.Record{ID: "", Source: "", Data: map[string]interface{}{}}
	close(in)

	validCh, errorCh := StartValidation(context.Background(), "job1", in, 2, nil)

	var validCount, errorCount int
	for range validCh {
		validCount++
	}
	for range errorCh {
		errorCount++
	}

	if validCount != 3 {
		t.Errorf("expected 3 valid records, got %d", validCount)
	}
	// record 4 → 1 error (missing id), record 5 → 2 errors (missing id + source) = 3 total
	if errorCount != 3 {
		t.Errorf("expected 3 validation errors, got %d", errorCount)
	}
	t.Logf("valid=%d  errors=%d", validCount, errorCount)
}
