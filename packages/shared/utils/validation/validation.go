package validation

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"pipeline/packages/shared/models"
)

var (
	validSourceTypes = map[string]bool{"csv": true, "json": true, "api": true}
	validExportTypes = map[string]bool{"json": true, "csv": true}
)

// ValidateJobSpec strictly checks a JobSpec before it's persisted or handed
// to the ingestion/export pipeline, so bad input fails fast with a 400
// instead of surfacing later as a DB row, a failed file write, or an
// outbound request to an arbitrary URL.
func ValidateJobSpec(spec models.JobSpec) error {
	if len(spec.Sources) == 0 {
		return errors.New("at least one source is required")
	}
	for i, src := range spec.Sources {
		if !validSourceTypes[src.Type] {
			return fmt.Errorf("sources[%d]: invalid type %q (must be csv, json, or api)", i, src.Type)
		}
		if err := ValidateSourceURL(src.URL); err != nil {
			return fmt.Errorf("sources[%d]: %w", i, err)
		}
	}

	if !validExportTypes[spec.Export.Type] {
		return fmt.Errorf("export: invalid type %q (must be json or csv)", spec.Export.Type)
	}
	if err := ValidateExportPath(spec.Export.Path); err != nil {
		return fmt.Errorf("export: %w", err)
	}

	c := spec.Concurrency
	if c.ValidationWorkers < 0 || c.TransformWorkers < 0 || c.IngestionBufferSize < 0 {
		return errors.New("concurrency: worker/buffer counts must not be negative")
	}
	if c.ValidationWorkers > MaxWorkers || c.TransformWorkers > MaxWorkers {
		return fmt.Errorf("concurrency: worker counts must not exceed %d", MaxWorkers)
	}
	if c.IngestionBufferSize > MaxIngestionBufferSize {
		return fmt.Errorf("concurrency: ingestion_buffer_size must not exceed %d", MaxIngestionBufferSize)
	}

	return nil
}

// ValidateSourceURL requires an absolute http(s) URL, since every reader
// (CSV, JSON, API) fetches the source over HTTP.
func ValidateSourceURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url must be an absolute http or https URL")
	}
	return nil
}

// ValidateExportPath rejects absolute paths and any ".." segment, so the
// exporter (which calls os.Create on this path verbatim) can't be made to
// write outside the working directory.
func ValidateExportPath(p string) error {
	if p == "" {
		return errors.New("path is required")
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return errors.New("path must be relative, not absolute")
	}
	for _, seg := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return errors.New("path must not contain '..'")
		}
	}
	return nil
}
