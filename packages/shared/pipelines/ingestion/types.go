package ingestion

import (
	"context"

	"pipeline/packages/shared/models"
)

// DataReader is the common interface all three readers satisfy.
type DataReader interface {
	Read(ctx context.Context, out chan<- models.Record) error
}

// CSVReader fetches a CSV file from a URL and streams each row as a Record.
type CSVReader struct{ URL string }

// JSONReader fetches a JSON array from a URL and streams each element as a Record.
type JSONReader struct{ URL string }

// APIReader fetches a single-object JSON response and flattens it into one Record.
// Designed for REST APIs like Open-Meteo that return one object, not an array.
type APIReader struct{ URL string }
