package store

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"pipeline/internal/models"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	godotenv.Load("../../.env")

	cfg := DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "pipeline_db"),
	}
	db, err := InitDB(cfg)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStore(db)
}

// newTestJob creates a job row and registers a cleanup to delete it after the test.
func newTestJob(t *testing.T, s *Store) string {
	t.Helper()
	id := uuid.New().String()
	job := models.PipelineJob{
		ID:        id,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
		Spec:      models.JobSpec{Concurrency: models.DefaultConcurrency()},
	}
	if err := s.CreateJob(job); err != nil {
		t.Fatalf("newTestJob: %v", err)
	}
	t.Cleanup(func() { s.DeleteJob(id) })
	return id
}

func TestCreateAndGetJob(t *testing.T) {
	s := setupTestStore(t)
	id := newTestJob(t, s)

	got, err := s.GetJob(id)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: want %s got %s", id, got.ID)
	}
	if got.Status != models.StatusPending {
		t.Errorf("Status: want pending, got %s", got.Status)
	}
	t.Logf("created and retrieved job %s", id)
}

func TestMarkStartedAndFinished(t *testing.T) {
	s := setupTestStore(t)
	id := newTestJob(t, s)

	if err := s.MarkStarted(id); err != nil {
		t.Fatalf("MarkStarted: %v", err)
	}
	job, err := s.GetJob(id)
	if err != nil {
		t.Fatalf("GetJob after MarkStarted: %v", err)
	}
	if job.Status != models.StatusRunning {
		t.Errorf("after MarkStarted: want running, got %s", job.Status)
	}
	if job.StartedAt == nil {
		t.Error("StartedAt should be set")
	}

	if err := s.MarkFinished(id, models.StatusCompleted, 150, 3); err != nil {
		t.Fatalf("MarkFinished: %v", err)
	}
	job, err = s.GetJob(id)
	if err != nil {
		t.Fatalf("GetJob after MarkFinished: %v", err)
	}
	if job.Status != models.StatusCompleted {
		t.Errorf("after MarkFinished: want completed, got %s", job.Status)
	}
	if job.RecordCount != 150 {
		t.Errorf("RecordCount: want 150, got %d", job.RecordCount)
	}
	if job.ErrorCount != 3 {
		t.Errorf("ErrorCount: want 3, got %d", job.ErrorCount)
	}
	t.Log("lifecycle: pending → running → completed")
}

func TestSaveAndGetErrors(t *testing.T) {
	s := setupTestStore(t)
	id := newTestJob(t, s)

	errs := []models.ValidationError{
		{JobID: id, RecordID: "r1", Field: "new_cases", Message: "must be a number", At: time.Now()},
		{JobID: id, RecordID: "r2", Field: "id", Message: "id is required", At: time.Now()},
	}
	for _, e := range errs {
		if err := s.SaveError(e); err != nil {
			t.Fatalf("SaveError: %v", err)
		}
	}

	got, err := s.GetErrors(id)
	if err != nil {
		t.Fatalf("GetErrors: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 errors, got %d", len(got))
	}
	t.Logf("saved and retrieved %d errors", len(got))
}

func TestSaveAndGetAggregation(t *testing.T) {
	s := setupTestStore(t)
	id := newTestJob(t, s)

	agg := models.AggregationResult{
		JobID:      id,
		TotalCount: 200,
		ValidCount: 195,
		ErrorCount: 5,
		BySource:   map[string]int{"csv": 200},
		NumericStats: map[string]models.FieldStats{
			"score": {Min: 10, Max: 90, Avg: 50, Sum: 10000, Count: 200},
		},
		ComputedAt: time.Now(),
	}

	if err := s.SaveAggregation(agg); err != nil {
		t.Fatalf("SaveAggregation: %v", err)
	}

	got, err := s.GetAggregation(id)
	if err != nil {
		t.Fatalf("GetAggregation: %v", err)
	}
	if got == nil {
		t.Fatal("expected aggregation result, got nil")
	}
	if got.TotalCount != 200 {
		t.Errorf("TotalCount: want 200, got %d", got.TotalCount)
	}
	if got.NumericStats["score"].Avg != 50 {
		t.Errorf("score avg: want 50, got %f", got.NumericStats["score"].Avg)
	}
	t.Logf("aggregation saved and retrieved — total=%d valid=%d", got.TotalCount, got.ValidCount)
}

func TestGetJob_NotFound(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.GetJob("non-existent-id")
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
	t.Log("correctly returned ErrJobNotFound")
}
