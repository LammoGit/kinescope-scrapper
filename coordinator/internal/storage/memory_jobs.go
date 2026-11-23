//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/domain"
	"fmt"
	"sync"
	"time"
)

// Won't compile if *JobRepository doesn't implement IJobRepository.
var _ domain.IJobRepository = &JobRepository{}

// JobRepository is an in-memory job store.
type JobRepository struct {
	jobs        map[int]domain.Job
	nextJobID   int
	nextRangeID int
	mu          sync.RWMutex
}

// NewJobRepository creates a new JobRepository.
func NewJobRepository() *JobRepository {
	return &JobRepository{
		jobs:        make(map[int]domain.Job),
		nextJobID:   1,
		nextRangeID: 1,
	}
}

// rangeTaken checks whether range collides with any of the jobs in repository.
// It's assumed that lock is already taken.
func (jr *JobRepository) rangeTaken(id int, from, to int) bool {
	for _, j := range jr.jobs {
		if j.Config.Range.From <= from && from <= j.Config.Range.To {
			return j.ID != id
		}
		if j.Config.Range.From <= to && to <= j.Config.Range.To {
			return j.ID != id
		}
	}
	return false
}

// SaveJob adds new job if id is 0 and updates otherwise.
func (jr *JobRepository) SaveJob(ctx context.Context, job *domain.Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	// Insert
	if job.ID == 0 {
		if jr.rangeTaken(0, job.Config.Range.From, job.Config.Range.To) {
			return domain.ErrRangeIsTaken
		}

		job.ID = jr.nextJobID
		jr.nextJobID++
		if job.Config.Name == "" {
			job.Config.Name = fmt.Sprintf("job-%d", job.ID)
		}
		jr.jobs[job.ID] = job.Clone()
		return nil
	}

	// Update
	if jr.rangeTaken(job.ID, job.Config.Range.From, job.Config.Range.To) {
		return domain.ErrRangeIsTaken
	}
	if _, ok := jr.jobs[job.ID]; !ok {
		return ErrJobNotFoundByID
	}

	jr.jobs[job.ID] = job.Clone()
	return nil
}

// GetJobByID returns a job by its identifier.
func (jr *JobRepository) GetJobByID(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	job, ok := jr.jobs[id]
	if ok {
		return job, nil
	}
	return job, ErrJobNotFoundByID
}

// GetJobs returns every stored job ordered by identifier
func (jr *JobRepository) GetJobs(ctx context.Context) ([]domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	ids := make([]int, 0, len(jr.jobs))
	for id := range jr.jobs {
		ids = append(ids, id)
	}

	res := make([]domain.Job, 0, len(ids))
	for _, id := range ids {
		res = append(res, jr.jobs[id])
	}
	return res, nil
}

// DeleteJobByID removes a job
func (jr *JobRepository) DeleteJobByID(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	if _, ok := jr.jobs[id]; !ok {
		return ErrJobNotFoundByID
	}

	delete(jr.jobs, id)
	return nil
}

// SaveRange adds new range if id is 0 and updates otherwise
func (jr *JobRepository) SaveRange(ctx context.Context, rangeBlock *domain.Range) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[rangeBlock.JobID]
	if !ok {
		return ErrJobNotFoundByID
	}

	// Insert
	if rangeBlock.ID == 0 {
		rangeBlock.ID = jr.nextRangeID
		jr.nextRangeID++
		now := time.Now()
		if rangeBlock.CreatedAt.IsZero() {
			rangeBlock.CreatedAt = now
		}
		if rangeBlock.Status == "" {
			rangeBlock.Status = domain.JobStatusCreated
		}
		rangeBlock.UpdatedAt = now
		rangeBlock.LeasedBy = 0
		if err := job.AddRange(rangeBlock.Clone()); err != nil {
			return err
		}
		jr.jobs[rangeBlock.JobID] = job
		return nil
	}

	// Update
	for idx, r := range job.Ranges {
		if r.ID == rangeBlock.ID {
			job.Ranges[idx] = rangeBlock.Clone()
			jr.jobs[rangeBlock.JobID] = job
			return nil
		}
	}
	return ErrRangeNotFoundByID
}

// GetRangeByID returns a copy of a range by its identifier
func (jr *JobRepository) GetRangeByID(ctx context.Context, id int) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	for _, job := range jr.jobs {
		for _, r := range job.Ranges {
			if r.ID == id {
				return r.Clone(), nil
			}
		}
	}
	return domain.Range{}, ErrRangeNotFoundByID
}

// GetRangesByJobID returns all ranges belonging to a job ordered by start
func (jr *JobRepository) GetRangesByJobID(ctx context.Context, jobID int) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	job, ok := jr.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFoundByID
	}

	var res []domain.Range
	for _, r := range job.Ranges {
		res = append(res, r.Clone())
	}
	return res, nil
}

// GetRangesByWorkerID
func (jr *JobRepository) GetRangesByWorkerID(ctx context.Context, workerID int) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	var res []domain.Range
	for _, job := range jr.jobs {
		for _, r := range job.Ranges {
			if r.LeasedBy == workerID {
				res = append(res, r.Clone())
			}
		}
	}
	return res, nil
}

// RenewLeases renews all ranges leased by a worker
func (jr *JobRepository) RenewLeases(ctx context.Context, workerID int, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	now := time.Now()
	for _, j := range jr.jobs {
		for i := range j.Ranges {
			r := &j.Ranges[i]
			if r.LeasedBy != workerID || r.Status != domain.JobStatusRunning || r.LeaseExpired(now) {
				continue
			}
			r.Renew(workerID, ttl)
		}
	}

	return nil
}

// DeleteRangesByJobID removes every range of a job
func (jr *JobRepository) DeleteRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[jobID]
	if !ok {
		return ErrJobNotFoundByID
	}

	job.DeleteRanges()
	jr.jobs[jobID] = job
	return nil
}
