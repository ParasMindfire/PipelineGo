package validation

const (
	// Upper bounds rejected by ValidateJobSpec to prevent a single job spec from
	// spinning up unbounded goroutines or channel buffers.
	MaxWorkers             = 100
	MaxIngestionBufferSize = 10000
)
