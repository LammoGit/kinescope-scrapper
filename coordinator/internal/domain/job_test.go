package domain_test

import (
	"coordinator/internal/domain"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func TestJob(t *testing.T) {
	now := time.Now()
	rangeBlock := domain.Range{
		ID:             12,
		JobID:          100,
		RangeStart:     0,
		RangeEnd:       100,
		NextID:         12,
		Status:         domain.JobStatusRunning,
		LeasedBy:       2,
		LeaseExpiresAt: now.Add(time.Hour),
		Attempt:        2,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	job := domain.Job{
		ID:         100,
		Config:     domain.JobConfig{},
		Workers:    []int{1, 2, 3},
		Ranges:     []domain.Range{rangeBlock},
		Status:     domain.JobStatusRunning,
		Paused:     false,
		Expiration: now.Add(time.Hour),
	}

	t.Run("Assign workers", func(t *testing.T) {
		clone := job.Clone()
		clone.AssignWorkers(2, 4, 5)

		if !slices.Contains(clone.Workers, 4) || !slices.Contains(clone.Workers, 5) {
			t.Fatalf("Job didn't assign passed workers: got %v", clone.Workers)
		}
		if len(clone.Workers) != 5 {
			t.Fatalf("expected 5 unique workers, got %v", clone.Workers)
		}
	})

	t.Run("Deassign specific workers", func(t *testing.T) {
		clone := job.Clone()

		// 99 doesn't exist, should ignore
		clone.DeassignWorkers(2, 99)

		if slices.Contains(clone.Workers, 2) || len(clone.Workers) != 2 {
			t.Fatalf("unexpected workers after deassign: got %v", clone.Workers)
		}
	})

	t.Run("Deassign all workers when none passed", func(t *testing.T) {
		clone := job.Clone()
		clone.DeassignWorkers()

		if len(clone.Workers) != 0 {
			t.Fatalf("expected all workers cleared, got %v", clone.Workers)
		}
	})

	t.Run("State transitions (Start, Pause, Resume, Stop)", func(t *testing.T) {
		clone := job.Clone()

		clone.Pause()
		if clone.Status != domain.JobStatusPaused || !clone.Paused {
			t.Fatalf("expected paused state, got status=%q paused=%v", clone.Status, clone.Paused)
		}

		clone.Resume()
		if clone.Status != domain.JobStatusRunning || clone.Paused {
			t.Fatalf("expected running state, got status=%q paused=%v", clone.Status, clone.Paused)
		}

		clone.Stop()
		if clone.Status != domain.JobStatusStopped || !clone.Paused || len(clone.Workers) != 0 || len(clone.Ranges) != 0 {
			t.Fatalf("expected stopped state with cleared workers & ranges, got %+v", clone)
		}

		clone.Start()
		if clone.Status != domain.JobStatusRunning || clone.Paused {
			t.Fatalf("expected running state, got status=%q paused=%v", clone.Status, clone.Paused)
		}
	})

	t.Run("AddRange", func(t *testing.T) {
		clone := job.Clone()
		clone.Config.Range = domain.JobRange{From: 0, To: 500}

		// Out of bounds
		err := clone.AddRange(domain.Range{ID: 13, RangeStart: 400, RangeEnd: 600})
		if !errors.Is(err, domain.ErrRangeIsOutsideOfJob) {
			t.Fatalf("expected ErrRangeIsOutsideOfJob, got %v", err)
		}

		// Overlaps existing [0, 100]
		err = clone.AddRange(domain.Range{ID: 14, RangeStart: 50, RangeEnd: 150})
		if !errors.Is(err, domain.ErrRangeIsTaken) {
			t.Fatalf("expected ErrRangeIsTaken, got %v", err)
		}

		// Valid non-overlapping range [101, 200]
		if err := clone.AddRange(domain.Range{ID: 15, RangeStart: 101, RangeEnd: 200}); err != nil {
			t.Fatalf("expected valid range to succeed, got %v", err)
		}
	})

	t.Run("DeleteRange", func(t *testing.T) {
		clone := job.Clone()

		clone.DeleteRange(0)
		if len(clone.Ranges) != len(job.Ranges) {
			t.Fatalf("unexpected ranges after deassign: got %v", clone.Ranges)
		}

		rid := clone.Ranges[0].ID
		clone.DeleteRange(rid)
		if slices.ContainsFunc(clone.Ranges, func(r domain.Range) bool {
			return r.ID == rid
		}) {
			t.Fatalf("unexpected ranges after deassign: got %v", clone.Ranges)
		}
		if len(clone.Ranges) != len(job.Ranges)-1 {
			t.Fatalf("unexpected ranges after deassign: got %v", clone.Ranges)
		}
	})

	t.Run("DeleteRanges", func(t *testing.T) {
		clone := job.Clone()

		clone.DeleteRanges(0)
		if len(clone.Ranges) != len(job.Ranges) {
			t.Fatalf("unexpected ranges after deassign: got %v", clone.Ranges)
		}

		clone.DeleteRanges()
		if len(clone.Ranges) != 0 {
			t.Fatalf("unexpected ranges after deassign: got %v", clone.Ranges)
		}
	})

	t.Run("Rejects claiming from a completed job", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusCompleted,
				Config: domain.JobConfig{Range: domain.JobRange{From: 1, To: 1000}},
			}

			_, err := job.ClaimNextRange(1, 30*time.Second)
			if !errors.Is(err, domain.ErrJobIsCompleted) {
				t.Fatalf("expected ErrJobIsCompleted, got %v", err)
			}
		})
	})

	t.Run("Carves new range from empty bounded job and respects maxSize", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			now := time.Now()
			ttl := 30 * time.Second
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusRunning,
				Config: domain.JobConfig{Range: domain.JobRange{From: 1, To: 5000}},
			}

			claimed, err := job.ClaimNextRange(10, ttl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if claimed.RangeStart != 1 || claimed.RangeEnd != 1000 || claimed.NextID != 1 {
				t.Fatalf("expected carved range [1, 1000] starting at 1, got [%d, %d] next=%d",
					claimed.RangeStart, claimed.RangeEnd, claimed.NextID)
			}
			if claimed.LeasedBy != 10 || claimed.Status != domain.JobStatusRunning || !claimed.LeaseExpiresAt.Equal(now.Add(ttl)) {
				t.Fatalf("unexpected lease metadata: %+v", claimed)
			}
			if len(job.Ranges) != 1 {
				t.Fatalf("expected 1 range stored on job, got %d", len(job.Ranges))
			}
		})
	})

	t.Run("Carves gap between existing ranges before advancing tail", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			now := time.Now()
			ttl := 30 * time.Second
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusRunning,
				Config: domain.JobConfig{Range: domain.JobRange{From: 1, To: 3000}},
				Ranges: []domain.Range{
					{ID: 1, JobID: 1, RangeStart: 1, RangeEnd: 500, Status: domain.JobStatusRunning, LeasedBy: 1, LeaseExpiresAt: now.Add(ttl)},
					{ID: 2, JobID: 1, RangeStart: 1001, RangeEnd: 2000, Status: domain.JobStatusCompleted, NextID: 2001},
				},
			}

			// Gap [501, 1000] is 500 items (<= maxSize 1000), so it should carve the whole gap
			claimed, err := job.ClaimNextRange(2, ttl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claimed.RangeStart != 501 || claimed.RangeEnd != 1000 {
				t.Fatalf("expected gap [501, 1000] to be carved first, got [%d, %d]", claimed.RangeStart, claimed.RangeEnd)
			}
		})
	})

	t.Run("Reclaims expired range before carving new ranges", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			ttl := 30 * time.Second
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusRunning,
				Config: domain.JobConfig{Range: domain.JobRange{From: 1, To: 5000}},
				Ranges: []domain.Range{
					{
						ID:             7,
						JobID:          1,
						RangeStart:     1,
						RangeEnd:       1000,
						NextID:         450,
						Status:         domain.JobStatusRunning,
						LeasedBy:       1,
						LeaseExpiresAt: start.Add(ttl),
					},
				},
			}

			time.Sleep(ttl + time.Second)

			claimed, err := job.ClaimNextRange(2, ttl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claimed.ID != 7 || claimed.LeasedBy != 2 || claimed.NextID != 450 {
				t.Fatalf("expected worker 2 to reclaim range 7 preserving NextID=450, got %+v", claimed)
			}
			if len(job.Ranges) != 1 {
				t.Fatalf("expected no new range carved when reclaiming expired range, got %d ranges", len(job.Ranges))
			}
		})
	})

	t.Run("Returns ErrRangeIsTaken when bounded job has no gaps or expired leases", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			now := time.Now()
			ttl := 30 * time.Second
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusRunning,
				Config: domain.JobConfig{Range: domain.JobRange{From: 1, To: 1000}},
				Ranges: []domain.Range{
					{ID: 1, JobID: 1, RangeStart: 1, RangeEnd: 1000, Status: domain.JobStatusRunning, LeasedBy: 1, LeaseExpiresAt: now.Add(ttl)},
				},
			}

			_, err := job.ClaimNextRange(2, ttl)
			if !errors.Is(err, domain.ErrRangeIsTaken) {
				t.Fatalf("expected ErrRangeIsTaken, got %v", err)
			}
		})
	})

	t.Run("Open-ended job (To == 0) keeps carving blocks past frontier", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ttl := 30 * time.Second
			job := domain.Job{
				ID:     1,
				Status: domain.JobStatusRunning,
				Config: domain.JobConfig{Range: domain.JobRange{From: 0, To: 0}},
			}

			r1, err := job.ClaimNextRange(1, ttl)
			if err != nil {
				t.Fatalf("first claim failed: %v", err)
			}
			r2, err := job.ClaimNextRange(2, ttl)
			if err != nil {
				t.Fatalf("second claim failed: %v", err)
			}

			if r1.RangeStart != 0 || r1.RangeEnd != 999 {
				t.Fatalf("expected first open block, got [%d, %d]", r1.RangeStart, r1.RangeEnd)
			}
			if r2.RangeStart != 1000 || r2.RangeEnd != 1999 {
				t.Fatalf("expected second open block, got [%d, %d]", r2.RangeStart, r2.RangeEnd)
			}
		})
	})
}

func TestRange(t *testing.T) {
	now := time.Now().UTC()
	base := domain.Range{
		ID:             12,
		JobID:          100,
		RangeStart:     0,
		RangeEnd:       100,
		NextID:         12,
		Status:         domain.JobStatusRunning,
		LeasedBy:       2,
		LeaseExpiresAt: now.Add(time.Hour),
		Attempt:        2,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	t.Run("Leased reports active worker assignment", func(t *testing.T) {
		r := base.Clone()
		if !r.Leased() {
			t.Fatal("expected range with LeasedBy=2 to report Leased() == true")
		}

		r.LeasedBy = 0
		if r.Leased() {
			t.Fatal("expected range with LeasedBy=0 to report Leased() == false")
		}
	})

	t.Run("LeaseExpired boundary checks", func(t *testing.T) {
		r := base.Clone()

		// Before expiration -> still valid
		if r.LeaseExpired(now.Add(59 * time.Minute)) {
			t.Fatal("expected lease to still be valid before LeaseExpiresAt")
		}

		// Exactly at expiration timestamp -> expired (!now.Before(LeaseExpiresAt))
		if !r.LeaseExpired(r.LeaseExpiresAt) {
			t.Fatal("expected lease to be expired at exact LeaseExpiresAt timestamp")
		}

		// After expiration -> expired
		if !r.LeaseExpired(now.Add(2 * time.Hour)) {
			t.Fatal("expected lease to be expired after LeaseExpiresAt")
		}

		// Unleased range -> always considered expired/available
		r.LeasedBy = 0
		if !r.LeaseExpired(now) {
			t.Fatal("expected unleased range to report LeaseExpired() == true")
		}
	})

	t.Run("Exhausted boundary checks", func(t *testing.T) {
		r := base.Clone()

		// NextID == RangeEnd means the last item still needs processing
		r.NextID = 100
		if r.Exhausted() {
			t.Fatal("expected range with NextID == RangeEnd not to be exhausted")
		}

		// NextID > RangeEnd means all items are done
		r.NextID = 101
		if !r.Exhausted() {
			t.Fatal("expected range with NextID > RangeEnd to be exhausted")
		}
	})

	t.Run("Lease assigns worker and rejects active lease by another worker", func(t *testing.T) {
		r := base.Clone()
		r.LeasedBy = 0
		r.LeaseExpiresAt = time.Time{}
		r.Status = domain.JobStatusCreated

		if err := r.Lease(7, 30*time.Minute); err != nil {
			t.Fatalf("expected unleased range to lease cleanly, got %v", err)
		}
		if r.LeasedBy != 7 || r.Status != domain.JobStatusRunning {
			t.Fatalf("unexpected state after Lease: got LeasedBy=%d Status=%q", r.LeasedBy, r.Status)
		}

		// Another worker trying to lease before expiration must fail
		if err := r.Lease(8, 30*time.Minute); !errors.Is(err, domain.ErrRangeIsTaken) {
			t.Fatalf("expected ErrRangeIsTaken when leasing active range, got %v", err)
		}
	})

	t.Run("Renew extends active lease for assigned worker", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			r := domain.Range{
				ID:             12,
				JobID:          100,
				RangeStart:     0,
				RangeEnd:       100,
				NextID:         12,
				Status:         domain.JobStatusRunning,
				LeasedBy:       2,
				LeaseExpiresAt: start.Add(10 * time.Minute),
				CreatedAt:      start,
				UpdatedAt:      start,
			}

			time.Sleep(5 * time.Minute)
			now := time.Now()

			if err := r.Renew(2, time.Hour); err != nil {
				t.Fatalf("expected valid lease renewal to succeed, got %v", err)
			}

			wantExpiry := now.Add(time.Hour)
			if !r.LeaseExpiresAt.Equal(wantExpiry) {
				t.Fatalf("got LeaseExpiresAt=%v, want %v", r.LeaseExpiresAt, wantExpiry)
			}
			if !r.UpdatedAt.Equal(now) {
				t.Fatalf("got UpdatedAt=%v, want %v", r.UpdatedAt, now)
			}
		})
	})

	t.Run("Renew rejects lease at exact expiration timestamp", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := domain.Range{
				LeasedBy:       2,
				LeaseExpiresAt: time.Now().Add(10 * time.Minute),
				Status:         domain.JobStatusRunning,
			}

			// Jump forward exactly 10 minutes
			time.Sleep(10 * time.Minute)

			if err := r.Renew(2, time.Hour); !errors.Is(err, domain.ErrWorkerDoesNotLeaseRange) {
				t.Fatalf("expected ErrWorkerDoesNotLeaseRange at exact expiration, got %v", err)
			}
		})
	})

	t.Run("Release clears lease and sets status based on Exhausted", func(t *testing.T) {
		// 1. Unfinished range returns to created
		r := base.Clone()
		r.Release()
		if r.LeasedBy != 0 || !r.LeaseExpiresAt.IsZero() || r.Status != domain.JobStatusCreated {
			t.Fatalf("expected unfinished released range to reset to created, got %+v", r)
		}

		// 2. Exhausted range transitions to completed on release
		done := base.Clone()
		done.NextID = done.RangeEnd + 1
		done.Release()
		if done.LeasedBy != 0 || done.Status != domain.JobStatusCompleted {
			t.Fatalf("expected exhausted released range to become completed, got %+v", done)
		}
	})

	t.Run("Complete clears lease and marks completed", func(t *testing.T) {
		r := base.Clone()
		r.Complete()

		if r.LeasedBy != 0 || !r.LeaseExpiresAt.IsZero() || r.Status != domain.JobStatusCompleted {
			t.Fatalf("expected completed range with cleared lease, got %+v", r)
		}
	})
}

// FuzzJobClone fuzzy job cloning tests
func FuzzJobClone(f *testing.F) {
	f.Add([]byte(`{
		"id": 1,
		"config": {
			"name": "scrape",
			"range": {"from": 1, "to": 1000},
			"servers": [{"host": "w1", "port": 8080}]
		},
		"workers":,
		"ranges": [{"id": 5, "job_id": 1, "range_start": 1, "range_end": 500, "leased_by": 10}],
		"status": "running",
		"paused": false
	}`))

	f.Add([]byte(`{"id": 2, "workers": [], "ranges": null}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Job
		if err := json.Unmarshal(data, &orig); err != nil {
			return // Skip invalid JSON bytes
		}

		clone := orig.Clone()

		// Deep-equality check
		if !reflect.DeepEqual(orig, clone) {
			t.Fatalf("clone mismatch:\norig:  %+v\nclone: %+v", orig, clone)
		}

		// Verify isolation
		if len(clone.Workers) > 0 {
			before := orig.Workers[0]
			clone.Workers[0] ^= 0x5555
			if orig.Workers[0] != before {
				t.Fatalf("Workers slice aliased: orig changed from %d to %d", before, orig.Workers)
			}
		}

		if len(clone.Ranges) > 0 {
			before := orig.Ranges[0].LeasedBy
			clone.Ranges[0].LeasedBy ^= 0x5555
			if orig.Ranges[0].LeasedBy != before {
				t.Fatalf("Ranges slice aliased: orig changed from %d to %d", before, orig.Ranges[0].LeasedBy)
			}
		}

		if len(clone.Config.Servers) > 0 {
			before := orig.Config.Servers
			clone.Config.Servers[0] = domain.WorkerConfig{}
			if !reflect.DeepEqual(orig.Config.Servers, before) {
				t.Fatalf("Config.Servers slice aliased: orig changed from %+v to %+v", before, orig.Config.Servers)
			}
		}
	})
}

// FuzzRangeClone fuzzy range cloning tests
func FuzzRangeClone(f *testing.F) {
	f.Add([]byte(`{
		"id": 10,
		"job_id": 1,
		"range_start": 1,
		"range_end": 1000,
		"next_id": 250,
		"status": "running",
		"leased_by": 42,
		"lease_expires_at": "2026-11-22T20:30:00Z",
		"attempt": 2,
		"created_at": "2026-11-22T20:00:00Z",
		"updated_at": "2026-11-22T20:15:00Z"
	}`))

	f.Add([]byte(`{"id": 1, "job_id": 1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Range
		if err := json.Unmarshal(data, &orig); err != nil {
			return // Skip invalid JSON bytes
		}

		clone := orig.Clone()

		// Deep-equality check
		if !reflect.DeepEqual(orig, clone) {
			t.Fatalf("clone mismatch:\norig:  %+v\nclone: %+v", orig, clone)
		}
	})
}
