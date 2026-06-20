package validation

import (
	"context"
	"sync"

	"pipeline/internal/models"
)

// StartValidation spins up numWorkers goroutines that read from in concurrently.
// Valid records go to validOut, validation errors go to errOut.
// Both channels close when all workers finish.
func StartValidation(
	ctx context.Context,
	jobID string,
	in <-chan models.Record,
	numWorkers int,
	progressCh chan<- models.ProgressEvent,
) (validOut <-chan models.Record, errOut <-chan models.ValidationError) {
	validCh := make(chan models.Record, 100)
	errorCh := make(chan models.ValidationError, 100)

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

				ok, errs := Validate(jobID, record)
				if ok {
					record.IsValid = true
					validCh <- record
					if progressCh != nil {
						progressCh <- models.ProgressEvent{Processed: 1}
					}
				} else {
					for _, e := range errs {
						errorCh <- e
					}
					if progressCh != nil {
						progressCh <- models.ProgressEvent{Errors: 1}
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(validCh)
		close(errorCh)
	}()

	return validCh, errorCh
}
