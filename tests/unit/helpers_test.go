//go:build unit

package unit_test

import "pipeline/packages/shared/models"

func makeRecord(id string, data map[string]interface{}) models.Record {
	return models.Record{
		ID:         id,
		Source:     "http://test.example.com",
		SourceType: "csv",
		Data:       data,
	}
}

func feedChannel(records []models.Record) <-chan models.Record {
	ch := make(chan models.Record, len(records))
	for _, r := range records {
		ch <- r
	}
	close(ch)
	return ch
}
