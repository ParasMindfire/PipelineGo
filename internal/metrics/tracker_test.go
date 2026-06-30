package metrics

import (
	"testing"
	"time"

	"pipeline/internal/models"
)

func TestTracker_Snapshot_InitialState(t *testing.T) {
	tr := NewTracker("job1", 100)

	snap := tr.Snapshot()

	if snap.JobID != "job1" {
		t.Errorf("JobID: want job1, got %s", snap.JobID)
	}
	if snap.Status != "running" {
		t.Errorf("Status: want running, got %s", snap.Status)
	}
	if snap.ProcessedCount != 0 {
		t.Errorf("ProcessedCount: want 0, got %d", snap.ProcessedCount)
	}
	if snap.EndTime != nil {
		t.Error("EndTime should be nil before SetFinished")
	}
}

func TestTracker_Listen_AccumulatesCounts(t *testing.T) {
	tr := NewTracker("job2", 0)
	progressCh := make(chan models.ProgressEvent, 10)

	done := make(chan struct{})
	go func() {
		tr.Listen(progressCh)
		close(done)
	}()

	progressCh <- models.ProgressEvent{Processed: 5}
	progressCh <- models.ProgressEvent{Processed: 3, Errors: 2}
	progressCh <- models.ProgressEvent{Errors: 1}
	close(progressCh)

	<-done

	snap := tr.Snapshot()
	if snap.ProcessedCount != 8 {
		t.Errorf("ProcessedCount: want 8, got %d", snap.ProcessedCount)
	}
	if snap.ErrorCount != 3 {
		t.Errorf("ErrorCount: want 3, got %d", snap.ErrorCount)
	}
}

func TestTracker_PercentComplete_KnownTotal(t *testing.T) {
	tr := NewTracker("job3", 200)
	progressCh := make(chan models.ProgressEvent, 5)

	go tr.Listen(progressCh)
	progressCh <- models.ProgressEvent{Processed: 100}
	close(progressCh)

	time.Sleep(10 * time.Millisecond) // let Listen drain

	snap := tr.Snapshot()
	if snap.PercentComplete != 50.0 {
		t.Errorf("PercentComplete: want 50.0, got %.1f", snap.PercentComplete)
	}
}

func TestTracker_PercentComplete_UnknownTotal(t *testing.T) {
	tr := NewTracker("job4", 0) // totalExpected=0 → unknown

	snap := tr.Snapshot()
	if snap.PercentComplete != -1.0 {
		t.Errorf("PercentComplete: want -1 for unknown total, got %.1f", snap.PercentComplete)
	}
}

func TestTracker_SetFinished(t *testing.T) {
	tr := NewTracker("job5", 0)

	tr.SetFinished("completed")
	snap := tr.Snapshot()

	if snap.Status != "completed" {
		t.Errorf("Status: want completed, got %s", snap.Status)
	}
	if snap.EndTime == nil {
		t.Error("EndTime should be set after SetFinished")
	}
}

func TestTracker_PercentComplete_CapsAt100(t *testing.T) {
	tr := NewTracker("job6", 10)
	progressCh := make(chan models.ProgressEvent, 5)

	go tr.Listen(progressCh)
	progressCh <- models.ProgressEvent{Processed: 999} // way over total
	close(progressCh)

	time.Sleep(10 * time.Millisecond)

	snap := tr.Snapshot()
	if snap.PercentComplete != 100.0 {
		t.Errorf("PercentComplete: want 100.0 (capped), got %.1f", snap.PercentComplete)
	}
}
