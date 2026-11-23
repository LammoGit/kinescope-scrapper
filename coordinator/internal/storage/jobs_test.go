package storage_test

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	crand "crypto/rand"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

func randomRange() domain.Range {
	return domain.Range{
		ID:             rand.Int(),
		JobID:          rand.Int(),
		RangeStart:     rand.Int(),
		RangeEnd:       rand.Int(),
		NextID:         rand.Int(),
		Status:         crand.Text(),
		LeasedBy:       rand.Int(),
		LeaseExpiresAt: time.UnixMilli(rand.Int64()),
		Attempt:        rand.Int(),
		CreatedAt:      time.UnixMilli(rand.Int64()),
		UpdatedAt:      time.UnixMilli(rand.Int64()),
	}
}

func randomJob() domain.Job {
	return domain.Job{
		ID: rand.Int(),
		Config: domain.JobConfig{
			Name: crand.Text(),
			Range: domain.JobRange{
				From: rand.Int(),
				To:   rand.Int(),
			},
			RequestTimeout:           rand.Int(),
			MaxTries:                 rand.Int(),
			GlobalRateLimitRPS:       rand.Int(),
			StopOnConsecutiveMissing: rand.Int(),
		},
		Workers:    rand.Perm(rand.IntN(10)),
		Ranges:     []domain.Range{},
		Status:     crand.Text(),
		Paused:     rand.Int()%2 == 0,
		Expiration: time.UnixMilli(rand.Int64()),
	}
}

func equalRanges(r1, r2 domain.Range) bool {
	return r1.ID == r2.ID &&
		r1.JobID == r2.JobID &&
		r1.RangeStart == r2.RangeStart &&
		r1.RangeEnd == r2.RangeEnd &&
		r1.NextID == r2.NextID &&
		r1.Status == r2.Status &&
		r1.LeasedBy == r2.LeasedBy &&
		r1.LeaseExpiresAt.Equal(r2.LeaseExpiresAt) &&
		r1.Attempt == r2.Attempt &&
		r1.CreatedAt.Equal(r2.CreatedAt) &&
		r1.UpdatedAt.Equal(r2.UpdatedAt)
}

func equalJobs(j1, j2 domain.Job) bool {
	return j1.ID == j2.ID &&
		j1.Config.Name == j2.Config.Name &&
		j1.Config.Range == j2.Config.Range &&
		j1.Config.Range.From == j2.Config.Range.From &&
		j1.Config.Range.To == j2.Config.Range.To &&
		j1.Config.RequestTimeout == j2.Config.RequestTimeout &&
		j1.Config.MaxTries == j2.Config.MaxTries &&
		j1.Config.GlobalRateLimitRPS == j2.Config.GlobalRateLimitRPS &&
		j1.Config.StopOnConsecutiveMissing == j2.Config.StopOnConsecutiveMissing &&
		slices.Equal(j1.Workers, j2.Workers) &&
		slices.EqualFunc(j1.Ranges, j2.Ranges, equalRanges) &&
		j1.Status == j2.Status &&
		j1.Paused == j2.Paused &&
		j1.Expiration.Equal(j2.Expiration)
}

func TestJobs(t *testing.T) {
	t.Run("Save and get a job", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		job := randomJob()
		job.Config.Name = ""
		job.ID = 0

		err = s.SaveJob(ctx, &job)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		{
			stored, err := s.GetJobs(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || !equalJobs(stored[0], job) {
				t.Fatalf("Received job isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetJobByID(ctx, job.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !equalJobs(stored, job) {
				t.Fatalf("Received job isn't equal to the saved one, got %v, want %v", stored, job)
			}
		}
	})

	t.Run("Save, update and get a job", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		job := randomJob()
		job.ID = 0

		err = s.SaveJob(ctx, &job)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		job2 := job.Clone()
		job2.Config.Range.From = job.Config.Range.To + 1
		job2.Config.Range.To = job2.Config.Range.From + 100
		job2.Status = job2.Status + "123"

		err = s.SaveJob(ctx, &job2)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		stored, err := s.GetJobByID(ctx, job.ID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if equalJobs(stored, job) {
			t.Fatalf("Received user is equal to the previous version, got %v, want %v", stored, job2)
		}

		if !equalJobs(stored, job2) {
			t.Fatalf("Received user isn't equal to either saved job, got %v, want %v", stored, job2)
		}
	})

	t.Run("Save, delete and get a job", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		job := randomJob()
		job.ID = 0

		err = s.SaveJob(ctx, &job)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		err = s.DeleteJobByID(ctx, job.ID)
		if err != nil {
			t.Fatalf("Failed to delete a job: %v", err)
		}

		{
			stored, err := s.GetJobs(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received job isn't equal to the saved one, got %v", stored)
			}
		}

		{
			_, err := s.GetJobByID(ctx, job.ID)
			if !errors.Is(err, storage.ErrJobNotFoundByID) {
				t.Fatalf("Unexpected error, got: %v, expected: %v", err, storage.ErrJobNotFoundByID)
			}
		}
	})
}

func TestRanges(t *testing.T) {
	t.Run("Save and get a range", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		job := randomJob()
		job.ID = 0
		job.Config.Range.From = 0
		job.Config.Range.To = 10000

		err = s.SaveJob(ctx, &job)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		r := randomRange()
		r.ID = 0
		r.JobID = job.ID
		r.RangeStart = 0
		r.RangeEnd = 100

		err = s.SaveRange(ctx, &r)
		if err != nil {
			t.Fatalf("Failed to save a range: %v", err)
		}

		{
			stored, err := s.GetRangesByJobID(ctx, r.JobID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || !equalRanges(stored[0], r) {
				t.Fatalf("Received range isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetRangesByWorkerID(ctx, r.LeasedBy)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || !equalRanges(stored[0], r) {
				t.Fatalf("Received range isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetRangeByID(ctx, r.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !equalRanges(stored, r) {
				t.Fatalf("Received range isn't equal to the saved one, got %v, want %v", stored, r)
			}
		}
	})

	t.Run("Save, delete and get a range", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		job := randomJob()
		job.ID = 0
		job.Config.Range.From = 0
		job.Config.Range.To = 10000

		err = s.SaveJob(ctx, &job)
		if err != nil {
			t.Fatalf("Failed to save a job: %v", err)
		}

		r := randomRange()
		r.ID = 0
		r.JobID = job.ID
		r.RangeStart = 0
		r.RangeEnd = 100

		err = s.SaveRange(ctx, &r)
		if err != nil {
			t.Fatalf("Failed to save a range: %v", err)
		}

		err = s.DeleteRangesByJobID(ctx, r.JobID)
		if err != nil {
			t.Fatalf("Failed to delete a range: %v", err)
		}

		{
			stored, err := s.GetRangesByJobID(ctx, r.JobID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received range isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetRangesByWorkerID(ctx, r.LeasedBy)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received range isn't equal to the saved one, got %v", stored)
			}
		}

		{
			_, err := s.GetRangeByID(ctx, r.ID)
			if !errors.Is(err, storage.ErrRangeNotFoundByID) {
				t.Fatalf("Unexpected error, got: %v, expected: %v", err, storage.ErrRangeNotFoundByID)
			}
		}
	})
}
