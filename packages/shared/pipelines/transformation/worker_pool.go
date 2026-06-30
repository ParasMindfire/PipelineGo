package transformation

import (
	"context"
	"sync"

	"pipeline/packages/shared/models"
)

// StartTransformation spins up numWorkers goroutines that read from in,
// transform each record, and send the result to the returned channel.
//
// No progress channel here: validation already reports every input record
// exactly once (as Processed or Errors), and Transform never fails, so this
// stage has nothing new to report. Reporting here too would double-count.
func StartTransformation(
	ctx context.Context,
	in <-chan models.Record,
	numWorkers int,
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
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
