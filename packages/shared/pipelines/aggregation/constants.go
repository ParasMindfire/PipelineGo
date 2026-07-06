package aggregation

const (
	// exportChannelBuffer is the capacity of the channel that carries records
	// from the aggregation fan-in to the export stage.
	exportChannelBuffer = 100

	// resultChannelBuffer holds the single AggregationResult emitted after all
	// records are processed. Buffered so the aggregation goroutine can send and
	// exit without waiting for the exporter to read.
	resultChannelBuffer = 1
)
