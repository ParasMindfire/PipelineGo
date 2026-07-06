package pipelines

const (
	// ProgressEventBuffer is the capacity of the channel that carries
	// ProgressEvent values from the validation stage to the tracker goroutine.
	// Sized larger than the record-channel buffers so progress reporting never
	// backs up the validation workers.
	ProgressEventBuffer = 500

	// DefaultOutputDir is the directory used when spec.Export.Path is empty.
	DefaultOutputDir = "data/output/"

	// DefaultExportType is the file format used for the export stage.
	DefaultExportType = "json"
)
