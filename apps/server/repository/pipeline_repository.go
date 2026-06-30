package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pipeline/packages/shared/models"
)

// ErrJobNotFound is returned by GetJob when no row matches the given ID.
var ErrJobNotFound = errors.New("job not found")

// NewPipelineRepository wraps an open *sql.DB.
func NewPipelineRepository(db *sql.DB) *PipelineRepository { return &PipelineRepository{db: db} }

// ── JOBS ──────────────────────────────────────────────────────────────────────

// CreateJob inserts a new job row with StatusPending.
func (s *PipelineRepository) CreateJob(job models.PipelineJob) error {
	specJSON, err := json.Marshal(job.Spec)
	if err != nil {
		return fmt.Errorf("create job: marshal spec: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO jobs (id, status, spec, created_at) VALUES ($1,$2,$3,$4)`,
		job.ID, int(job.Status), specJSON, job.CreatedAt,
	)
	return err
}

// GetJob fetches a single job by ID. Returns ErrJobNotFound if no row exists.
func (s *PipelineRepository) GetJob(id string) (*models.PipelineJob, error) {
	var job models.PipelineJob
	var statusInt int
	var specJSON []byte

	err := s.db.QueryRow(
		`SELECT id, status, spec, created_at, started_at, finished_at, error_count, record_count
		 FROM jobs WHERE id = $1`, id,
	).Scan(
		&job.ID, &statusInt, &specJSON,
		&job.CreatedAt, &job.StartedAt, &job.FinishedAt,
		&job.ErrorCount, &job.RecordCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	job.Status = models.JobStatus(statusInt)
	if err := json.Unmarshal(specJSON, &job.Spec); err != nil {
		return nil, fmt.Errorf("get job: unmarshal spec: %w", err)
	}
	return &job, nil
}

// ListJobs returns all jobs ordered by creation time descending.
func (s *PipelineRepository) ListJobs() ([]models.PipelineJob, error) {
	rows, err := s.db.Query(
		`SELECT id, status, created_at, error_count, record_count
		 FROM jobs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.PipelineJob
	for rows.Next() {
		var job models.PipelineJob
		var statusInt int
		if err := rows.Scan(&job.ID, &statusInt, &job.CreatedAt, &job.ErrorCount, &job.RecordCount); err != nil {
			return nil, fmt.Errorf("list jobs: scan row: %w", err)
		}
		job.Status = models.JobStatus(statusInt)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// MarkStarted sets status=running and records the start timestamp.
func (s *PipelineRepository) MarkStarted(id string) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE jobs SET status=$1, started_at=$2 WHERE id=$3`,
		int(models.StatusRunning), now, id,
	)
	return err
}

// MarkFinished sets the final status, finish timestamp, and counts.
func (s *PipelineRepository) MarkFinished(id string, status models.JobStatus, recordCount, errorCount int) error {
	now := time.Now()
	_, err := s.db.Exec(
		`UPDATE jobs SET status=$1, finished_at=$2, record_count=$3, error_count=$4 WHERE id=$5`,
		int(status), now, recordCount, errorCount, id,
	)
	return err
}

// UpdateStatus updates only the status column.
func (s *PipelineRepository) UpdateStatus(id string, status models.JobStatus) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status=$1 WHERE id=$2`,
		int(status), id,
	)
	return err
}

// DeleteJob removes a job and cascades to job_errors and aggregation_results.
func (s *PipelineRepository) DeleteJob(id string) error {
	_, err := s.db.Exec(`DELETE FROM jobs WHERE id=$1`, id)
	return err
}

// CountsByStatus aggregates job counts per status in SQL, so callers (e.g.
// the /metrics endpoint, scraped every few seconds) don't have to fetch and
// hydrate every job row just to tally them in Go.
func (s *PipelineRepository) CountsByStatus() (map[models.JobStatus]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[models.JobStatus]int)
	for rows.Next() {
		var statusInt, count int
		if err := rows.Scan(&statusInt, &count); err != nil {
			return nil, fmt.Errorf("counts by status: scan row: %w", err)
		}
		counts[models.JobStatus(statusInt)] = count
	}
	return counts, rows.Err()
}

// ── ERRORS ────────────────────────────────────────────────────────────────────

// SaveError persists one ValidationError to job_errors.
func (s *PipelineRepository) SaveError(e models.ValidationError) error {
	_, err := s.db.Exec(
		`INSERT INTO job_errors (job_id, record_id, field, message, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		e.JobID, e.RecordID, e.Field, e.Message, e.At,
	)
	return err
}

// GetErrors returns all validation errors for a job, ordered by time.
func (s *PipelineRepository) GetErrors(jobID string) ([]models.ValidationError, error) {
	rows, err := s.db.Query(
		`SELECT job_id, record_id, field, message, created_at
		 FROM job_errors WHERE job_id=$1 ORDER BY created_at`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errs []models.ValidationError
	for rows.Next() {
		var e models.ValidationError
		if err := rows.Scan(&e.JobID, &e.RecordID, &e.Field, &e.Message, &e.At); err != nil {
			return nil, fmt.Errorf("get errors: scan row: %w", err)
		}
		errs = append(errs, e)
	}
	return errs, rows.Err()
}

// ── AGGREGATION ───────────────────────────────────────────────────────────────

// SaveAggregation upserts the final aggregation result for a job.
func (s *PipelineRepository) SaveAggregation(r models.AggregationResult) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("save aggregation: marshal: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO aggregation_results (job_id, data, computed_at) VALUES ($1,$2,$3)
		 ON CONFLICT (job_id) DO UPDATE SET data=$2, computed_at=$3`,
		r.JobID, data, r.ComputedAt,
	)
	return err
}

// GetAggregation returns the aggregation result for a job, or nil if not yet computed.
func (s *PipelineRepository) GetAggregation(jobID string) (*models.AggregationResult, error) {
	var data []byte
	err := s.db.QueryRow(
		`SELECT data FROM aggregation_results WHERE job_id=$1`, jobID,
	).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get aggregation: %w", err)
	}
	var r models.AggregationResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("get aggregation: unmarshal: %w", err)
	}
	return &r, nil
}
