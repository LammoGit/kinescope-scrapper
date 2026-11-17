package repos

import (
	"context"
	"coordinator/internal/domain"
	"testing"
	"time"
)

// claimJob builds a job and returns its repository
func claimJob(t *testing.T, jr *JobRepository, cfg domain.JobConfig) domain.Job {
	t.Helper()
	job, err := jr.CreateJob(context.Background(), "test", cfg)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	return job
}

// TestTryClaimRangeOpenEnded checks that an open-ended job keeps serving blocks
// past the frontier of every already carved block
func TestTryClaimRangeOpenEnded(t *testing.T) {
	jr := NewJobRepository()
	job := claimJob(t, jr, domain.JobConfig{
		Range: domain.JobRange{From: 0, To: 0},
	})

	var lastEnd = job.Config.Range.From - 1
	for i := 0; i < 3; i++ {
		r, err := jr.TryClaimRange(context.Background(), i+1, job.ID, time.Minute)
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		if r.RangeStart != lastEnd+1 {
			t.Fatalf("claim %d: start %d, want %d", i, r.RangeStart, lastEnd+1)
		}
		if r.RangeEnd <= r.RangeStart {
			t.Fatalf("claim %d: unbounded block %d-%d", i, r.RangeStart, r.RangeEnd)
		}
		if r.RangeEnd-r.RangeStart+1 > 1000 {
			t.Fatalf("claim %d: block too large %d", i, r.RangeEnd-r.RangeStart+1)
		}
		lastEnd = r.RangeEnd

		// Complete the block so the next claim has to carve a new one
		if err := jr.CompleteRange(context.Background(), r.ID); err != nil {
			t.Fatalf("complete %d: %v", i, err)
		}
	}
}

// TestTryClaimRangeFiniteStillWorks checks the finite range behavior is unchanged
func TestTryClaimRangeFiniteStillWorks(t *testing.T) {
	jr := NewJobRepository()
	job := claimJob(t, jr, domain.JobConfig{
		Range: domain.JobRange{From: 5, To: 14},
	})

	seen := 0
	for {
		r, err := jr.TryClaimRange(context.Background(), 1, job.ID, time.Minute)
		if err != nil {
			break
		}
		seen++
		if r.RangeStart < 5 || r.RangeEnd > 14 {
			t.Fatalf("block %d-%d outside of the job range", r.RangeStart, r.RangeEnd)
		}
		if err := jr.CompleteRange(context.Background(), r.ID); err != nil {
			t.Fatalf("complete: %v", err)
		}
	}
	if seen == 0 {
		t.Fatal("no blocks were served")
	}

	// The range is exhausted, a new claim must fail
	if _, err := jr.TryClaimRange(context.Background(), 1, job.ID, time.Minute); err == nil {
		t.Fatal("claim succeeded on an exhausted range")
	}
}
