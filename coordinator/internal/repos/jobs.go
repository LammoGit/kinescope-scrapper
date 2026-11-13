package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Won't compile if *JobRepository doesn't implement IJobRepository
var _ IJobRepository = &JobRepository{}

// Errors
var (
	ErrJobNotFoundByID   = utility.NewNotFoundError("job with given id doesn't exist")
	ErrRangeNotFoundByID = utility.NewNotFoundError("range with given id doesn't exist.")
	ErrRangeIsCompleted  = utility.NewConflictError("range is already completed")
	ErrRangeIsTaken      = utility.NewConflictError("range is already taken")
)

// IJobRepository stores scraping jobs and their state
type IJobRepository interface {
	// Jobs
	CreateJob(ctx context.Context, name string, settings domain.JobConfig) (domain.Job, error)
	AssignJob(ctx context.Context, id int, worker domain.Worker) error
	GetJob(ctx context.Context, id int) (domain.Job, error)
	GetJobs(ctx context.Context) ([]domain.Job, error)
	DeleteJob(ctx context.Context, id int) error
	StartJob(ctx context.Context, id int) error
	PauseJob(ctx context.Context, id int) error
	ResumeJob(ctx context.Context, id int) error
	StopJob(ctx context.Context, id int) error
	ReplaceJob(ctx context.Context, job domain.Job) error

	// Ranges
	AddRange(ctx context.Context, rangeBlock domain.Range) error
	GetRangeByID(ctx context.Context, id int) (domain.Range, error)
	GetRangesByJobID(ctx context.Context, jobID int) ([]domain.Range, error)
	GetUnassignedRanges(ctx context.Context) ([]domain.Range, error)
	TryClaimRange(ctx context.Context, rangeID, workerID int, leaseTTL time.Duration) error
	RenewLeases(ctx context.Context, workerID int, leaseTTL time.Duration) (int, error)
	ReleaseRangeByID(ctx context.Context, rangeID int) error
	ReleaseRangesByJobID(ctx context.Context, jobID int) error
	CompleteRange(ctx context.Context, rangeID int) error
	DeleteRangesByJobID(ctx context.Context, jobID int) error
}

// JobRepository is an in-memory job store
type JobRepository struct {
	jobs        map[int]domain.Job
	ranges      map[int]domain.Range
	jobRanges   map[int]map[int]struct{}
	nextJobID   int
	nextRangeID int
	mu          sync.RWMutex
}

// NewJobRepository creates a new JobRepository
func NewJobRepository() *JobRepository {
	return &JobRepository{
		jobs:        make(map[int]domain.Job),
		ranges:      make(map[int]domain.Range),
		jobRanges:   make(map[int]map[int]struct{}),
		nextJobID:   1,
		nextRangeID: 1,
	}
}

// CreateJob creates and stores a new job
func (jr *JobRepository) CreateJob(ctx context.Context, name string, config domain.JobConfig) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	jr.nextJobID++
	id := jr.nextJobID

	if config.Name == "" {
		config.Name = fmt.Sprintf("job-%d", id)
	}

	job := domain.Job{
		ID:     id,
		Config: config,
		Status: domain.JobStatusCreated,
	}
	jr.jobs[id] = job

	return job, nil
}

// AssignJob attaches a worker to the given job
func (jr *JobRepository) AssignJob(ctx context.Context, id int, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return ErrJobNotFoundByID
	}

	job.AssignWorkers(worker.ID)

	jr.jobs[id] = job
	return nil
}

// GetJob returns a job by its identifier
func (jr *JobRepository) GetJob(ctx context.Context, id int) (domain.Job, error) {
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
	sort.Ints(ids)

	res := make([]domain.Job, 0, len(ids))
	for _, id := range ids {
		res = append(res, jr.jobs[id])
	}
	return res, nil
}

// DeleteJob removes a job
func (jr *JobRepository) DeleteJob(ctx context.Context, id int) error {
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

// StartJob moves a job into the running state
func (jr *JobRepository) StartJob(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return nil
}

// PauseJob moves a job into the paused state
func (jr *JobRepository) PauseJob(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusPaused
	job.Paused = true
	jr.jobs[id] = job
	return nil
}

// ResumeJob moves a paused job back into the running state
func (jr *JobRepository) ResumeJob(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return nil
}

// StopJob moves a job into the stopped state
func (jr *JobRepository) StopJob(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {

		return ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusStopped
	job.Paused = true
	job.DeassignWorkers()
	jr.jobs[id] = job
	return nil
}

// ReplaceJob replaces a stored job
func (jr *JobRepository) ReplaceJob(ctx context.Context, job domain.Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	if _, ok := jr.jobs[job.ID]; !ok {
		return ErrJobNotFoundByID
	}

	jr.jobs[job.ID] = job
	return nil
}

// AddRange stores a range, assigning it an identifier when needed
func (jr *JobRepository) AddRange(ctx context.Context, rangeBlock domain.Range) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	if rangeBlock.ID == 0 {
		jr.nextRangeID++
		rangeBlock.ID = jr.nextRangeID
	} else if rangeBlock.ID > jr.nextRangeID {
		jr.nextRangeID = rangeBlock.ID
	}

	now := time.Now()
	if rangeBlock.CreatedAt.IsZero() {
		rangeBlock.CreatedAt = now
	}
	if rangeBlock.NextID == 0 {
		rangeBlock.NextID = rangeBlock.RangeStart
	}
	if rangeBlock.Status == "" {
		rangeBlock.Status = domain.JobStatusCreated
	}
	rangeBlock.UpdatedAt = now

	jr.ranges[rangeBlock.ID] = rangeBlock
	return nil
}

// GetRangeByID returns a copy of a range by its identifier
func (jr *JobRepository) GetRangeByID(ctx context.Context, id int) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	r, ok := jr.ranges[id]
	if !ok {
		return r, ErrRangeNotFoundByID
	}
	return r, nil
}

// GetRangesByJobID returns all ranges belonging to a job ordered by start
func (jr *JobRepository) GetRangesByJobID(ctx context.Context, jobID int) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	var res []domain.Range
	for _, r := range jr.ranges {
		if r.JobID == jobID {
			res = append(res, r)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].RangeStart == res[j].RangeStart {
			return res[i].ID < res[j].ID
		}
		return res[i].RangeStart < res[j].RangeStart
	})
	return res, nil
}

// GetUnassignedRanges returns ranges that are ready to be leased
func (jr *JobRepository) GetUnassignedRanges(ctx context.Context) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jr.mu.RLock()
	defer jr.mu.RUnlock()

	now := time.Now()
	var res []domain.Range
	for _, r := range jr.ranges {
		if r.Status == domain.JobStatusCompleted || r.Status == domain.JobStatusStopped || r.Status == domain.JobStatusFailed {
			continue
		}
		if !r.Leased() || r.LeaseExpired(now) {
			res = append(res, r)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].JobID == res[j].JobID {
			return res[i].RangeStart < res[j].RangeStart
		}
		return res[i].JobID < res[j].JobID
	})
	return res, nil
}

// TryClaimRange attempts to lease a range to a worker
func (jr *JobRepository) TryClaimRange(ctx context.Context, rangeID, workerID int, leaseTTL time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	r, ok := jr.ranges[rangeID]
	if !ok {
		return ErrRangeNotFoundByID
	}
	if r.Status == domain.JobStatusCompleted || r.Status == domain.JobStatusStopped || r.Status == domain.JobStatusFailed {
		return ErrRangeIsCompleted
	}

	now := time.Now()
	if r.Leased() && !r.LeaseExpired(now) {
		return ErrRangeIsTaken
	}

	r.LeasedBy = workerID
	r.LeaseExpiresAt = now.Add(leaseTTL)
	r.Status = domain.JobStatusRunning
	r.UpdatedAt = now
	jr.ranges[r.ID] = r
	return nil
}

// RenewLeases extends every active lease of a worker
func (jr *JobRepository) RenewLeases(ctx context.Context, workerID int, leaseTTL time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	now := time.Now()
	renewed := 0
	for id, r := range jr.ranges {
		if r.LeasedBy != workerID || r.Status != domain.JobStatusRunning {
			continue
		}
		r.LeaseExpiresAt = now.Add(leaseTTL)
		r.UpdatedAt = now
		jr.ranges[id] = r
		renewed++
	}
	return renewed, nil
}

// ReleaseRangeByID unassigns a range from its worker while keeping its progress
func (jr *JobRepository) ReleaseRangeByID(ctx context.Context, rangeID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	r, ok := jr.ranges[rangeID]
	if !ok {
		return ErrRangeNotFoundByID
	}

	now := time.Now()
	r.LeasedBy = 0
	r.LeaseExpiresAt = time.Time{}
	r.UpdatedAt = now
	if r.Exhausted() {
		r.Status = domain.JobStatusCompleted
	} else {
		r.Status = domain.JobStatusCreated
	}
	jr.ranges[r.ID] = r
	return nil
}

// ReleaseRangesByJobID unassigns ranges by their job id from their workers while keeping their progress
func (jr *JobRepository) ReleaseRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	ranges, ok := jr.jobRanges[jobID]
	if !ok {
		return ErrJobNotFoundByID
	}

	for rid := range ranges {
		r, ok := jr.ranges[rid]
		if !ok {
			continue
		}
		now := time.Now()
		r.LeasedBy = 0
		r.LeaseExpiresAt = time.Time{}
		r.UpdatedAt = now
		if r.Exhausted() {
			r.Status = domain.JobStatusCompleted
		} else {
			r.Status = domain.JobStatusCreated
		}
		jr.ranges[r.ID] = r
	}

	return nil
}

// CompleteRange marks a range as fully processed
func (jr *JobRepository) CompleteRange(ctx context.Context, rangeID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	r, ok := jr.ranges[rangeID]
	if !ok {
		return ErrRangeNotFoundByID
	}

	r.LeasedBy = 0
	r.LeaseExpiresAt = time.Time{}
	r.Status = domain.JobStatusCompleted
	r.UpdatedAt = time.Now()
	jr.ranges[r.ID] = r
	return nil
}

// DeleteRangesByJobID removes every range of a job
func (jr *JobRepository) DeleteRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	for id, r := range jr.ranges {
		if r.JobID == jobID {
			delete(jr.ranges, id)
		}
	}

	return nil
}
