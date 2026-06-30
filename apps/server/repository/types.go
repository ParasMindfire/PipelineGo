package repository

import "database/sql"

// PipelineRepository is the single point of access to the database.
// All SQL lives here — no SQL in controllers, services, or the pipeline runner.
type PipelineRepository struct{ db *sql.DB }
