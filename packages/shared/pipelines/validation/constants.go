package validation

const (
	// channelBuffer is the capacity of the valid-records and error channels
	// produced by StartValidation. Buffering decouples validation workers from
	// the downstream transformation stage so neither blocks the other.
	channelBuffer = 100
)
