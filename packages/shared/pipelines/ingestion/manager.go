package ingestion

import (
	"context"
	"log"
	"sync"

	"pipeline/packages/shared/models"
)

func newReader(src models.SourceConfig) DataReader {
	switch src.Type {
	case "csv":
		return &CSVReader{URL: src.URL}
	case "json":
		return &JSONReader{URL: src.URL}
	case "api":
		return &APIReader{URL: src.URL}
	default:
		log.Printf("ingestion: skipping unknown source type %q", src.Type)
		return nil
	}
}

// StartIngestion fans out across all sources concurrently.
// Each source runs in its own goroutine. The returned channel closes
// when ALL readers have finished or failed.
func StartIngestion(
	ctx context.Context,
	sources []models.SourceConfig,
	bufferSize int,
) <-chan models.Record {
	out := make(chan models.Record, bufferSize)
	var wg sync.WaitGroup

	for _, src := range sources {
		reader := newReader(src)
		if reader == nil {
			continue
		}

		wg.Add(1)
		go func(r DataReader, srcURL string) {
			defer wg.Done()
			if err := r.Read(ctx, out); err != nil {
				log.Printf("ingestion: reader failed for %s: %v", srcURL, err)
			}
		}(reader, src.URL)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
