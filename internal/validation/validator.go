package validation

import (
	"fmt"
	"strconv"
	"time"

	"pipeline/internal/models"
)

// Validate checks all rules on a record and returns ALL errors found, not just the first.
func Validate(jobID string, r models.Record) (bool, []models.ValidationError) {
	var errs []models.ValidationError

	add := func(field, msg string) {
		errs = append(errs, models.ValidationError{
			JobID:    jobID,
			RecordID: r.ID,
			Field:    field,
			Message:  msg,
			At:       time.Now(),
		})
	}

	if r.ID == "" {
		add("id", "id is required")
	}

	if r.Source == "" {
		add("source", "source is required")
	}

	numericFields := []string{
		"new_cases", "new_deaths", "cases", "deaths",
		"height(inches)", "weight(pounds)",
		"price", "temperature", "windspeed",
	}
	for _, field := range numericFields {
		if val, ok := r.Data[field]; ok {
			if str, isStr := val.(string); isStr && str != "" {
				if _, err := strconv.ParseFloat(str, 64); err != nil {
					add(field, fmt.Sprintf("%s must be a number, got %q", field, str))
				}
			}
		}
	}

	return len(errs) == 0, errs
}
