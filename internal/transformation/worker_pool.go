package transformation

import (
	"context"
	"sync"

	"pipeline/internal/models"
)

// StartTransformation spins up numWorkers goroutines that read from in,
// transform each record, and send the result to the returned channel.
// Progress events are reported via progressCh if non-nil.
func StartTransformation(
	ctx context.Context,
	in <-chan models.Record,
	numWorkers int,
	progressCh chan<- models.ProgressEvent,
) <-chan models.Record {
	out := make(chan models.Record, 100)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range in {
				select {
				case <-ctx.Done():
					return
				default:
				}
				out <- Transform(record)
				if progressCh != nil {
					progressCh <- models.ProgressEvent{Processed: 1}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
