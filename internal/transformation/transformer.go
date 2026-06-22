package transformation

import (
	"strconv"
	"strings"
	"time"

	"pipeline/internal/models"
)

// Transform cleans and normalizes a record's Data fields.
// Strings that parse as numbers become float64.
// Other strings are trimmed and lowercased.
// Non-string values (float64, bool, etc.) pass through unchanged.
func Transform(r models.Record) models.Record {
	now := time.Now()
	r.ProcessedAt = &now

	for key, val := range r.Data {
		str, ok := val.(string)
		if !ok {
			continue
		}

		str = strings.TrimSpace(str)

		if f, err := strconv.ParseFloat(str, 64); err == nil {
			r.Data[key] = f
		} else {
			r.Data[key] = strings.ToLower(str)
		}
	}

	return r
}
