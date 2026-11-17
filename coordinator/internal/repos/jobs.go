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
	ErrRangeNotFoundByID = utility.NewNotFoundError("range with given id doesn't exist")
	ErrRangeIsCompleted  = utility.NewConflictError("range is already completed")
	ErrRangeIsTaken      = utility.NewConflictError("range is already taken")
	ErrJobIsCompleted    = utility.NewConflictError("job is already completed")
)

// IJobRepository stores scraping jobs and their state
type IJobRepository interface {
	// Jobs
	CreateJob(ctx context.Context, name string, settings domain.JobConfig) (domain.Job, error)
	AssignJob(ctx context.Context, id int, worker domain.Worker) error
	GetJob(ctx context.Context, id int) (domain.Job, error)
	GetJobs(ctx context.Context) ([]domain.Job, error)
	DeleteJob(ctx context.Context, id int) (domain.Job, error)
	StartJob(ctx context.Context, id int) (domain.Job, error)
	PauseJob(ctx context.Context, id int) (domain.Job, error)
	ResumeJob(ctx context.Context, id int) (domain.Job, error)
	StopJob(ctx context.Context, id int) (domain.Job, error)
	ReplaceJob(ctx context.Context, job domain.Job) error

	// Ranges
	AddRange(ctx context.Context, rangeBlock domain.Range) error
	GetRangeByID(ctx context.Context, id int) (domain.Range, error)
	GetRangesByJobID(ctx context.Context, jobID int) ([]domain.Range, error)
	TryClaimRange(ctx context.Context, workerID, jobID int, leaseTTL time.Duration) (domain.Range, error)
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
	jr.jobRanges[id] = make(map[int]struct{})

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
func (jr *JobRepository) DeleteJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	var job domain.Job
	var ok bool
	if job, ok = jr.jobs[id]; !ok {
		return job, ErrJobNotFoundByID
	}

	// Cascade
	for rangeID, r := range jr.ranges {
		if r.JobID == id {
			delete(jr.ranges, rangeID)
		}
	}

	delete(jr.jobs, id)
	delete(jr.jobRanges, id)
	return job, nil
}

// StartJob moves a job into the running state
func (jr *JobRepository) StartJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return job, ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return job, nil
}

// PauseJob moves a job into the paused state
func (jr *JobRepository) PauseJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return job, ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusPaused
	job.Paused = true
	jr.jobs[id] = job
	return job, nil
}

// ResumeJob moves a paused job back into the running state
func (jr *JobRepository) ResumeJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {
		return job, ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusRunning
	job.Paused = false
	jr.jobs[id] = job
	return job, nil
}

// StopJob moves a job into the stopped state
func (jr *JobRepository) StopJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[id]
	if !ok {

		return job, ErrJobNotFoundByID
	}

	job.Status = domain.JobStatusStopped
	job.Paused = true
	job.DeassignWorkers()
	jr.jobs[id] = job
	for rid := range jr.jobRanges[id] {
		delete(jr.jobRanges, rid)
	}
	return job, nil
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

	if m, ok := jr.jobRanges[rangeBlock.JobID]; ok {
		m[rangeBlock.ID] = struct{}{}
	} else {
		return ErrJobNotFoundByID
	}

	for _, r := range jr.ranges {
		if r.RangeStart >= rangeBlock.RangeStart && rangeBlock.RangeStart <= r.RangeEnd {
			return ErrRangeIsTaken
		}
		if r.RangeStart >= rangeBlock.RangeEnd && rangeBlock.RangeEnd <= r.RangeEnd {
			return ErrRangeIsTaken
		}
	}

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

// TryClaimRange attempts to lease a range to a worker
func (jr *JobRepository) TryClaimRange(ctx context.Context, workerID, jobID int, leaseTTL time.Duration) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	jr.mu.Lock()
	defer jr.mu.Unlock()

	job, ok := jr.jobs[jobID]
	if !ok {
		return domain.Range{}, ErrJobNotFoundByID
	}
	if job.Status == domain.JobStatusCompleted {
		return domain.Range{}, ErrJobIsCompleted
	}

	now := time.Now()

	owned := make([]domain.Range, 0)
	for _, r := range jr.ranges {
		if r.JobID == jobID {
			owned = append(owned, r)
		}
	}
	sort.Slice(owned, func(i, j int) bool {
		if owned[i].RangeStart == owned[j].RangeStart {
			return owned[i].ID < owned[j].ID
		}
		return owned[i].RangeStart < owned[j].RangeStart
	})

	for _, r := range owned {
		if r.Status == domain.JobStatusCompleted || r.Exhausted() {
			continue
		}
		if !r.LeaseExpired(now) {
			continue
		}
		r.LeasedBy = workerID
		r.LeaseExpiresAt = now.Add(leaseTTL)
		r.Status = domain.JobStatusRunning
		r.UpdatedAt = now
		jr.ranges[r.ID] = r
		return r, nil
	}

	cursor := job.Config.Range.From
	gapEnd := job.Config.Range.To
	maxSize := 1000
	open := job.Config.Range.To <= 0
	tail := true
	for _, r := range owned {
		if r.RangeStart > cursor {
			gapEnd = r.RangeStart - 1
			tail = false
			break
		}
		if r.RangeEnd >= cursor {
			cursor = r.RangeEnd + 1
		}
	}
	if open && tail {
		// Open-ended jobs keep carving full blocks past the frontier
		gapEnd = cursor + maxSize - 1
	} else if !open && cursor > job.Config.Range.To {
		return domain.Range{}, ErrRangeIsTaken
	}

	gapSize := gapEnd - cursor + 1
	var size int
	switch {
	case gapSize <= maxSize:
		size = gapSize
	case gapSize < 2*maxSize:
		size = gapSize / 2
	default:
		size = maxSize
	}
	end := cursor + size - 1

	jr.nextRangeID++
	carved := domain.Range{
		ID:             jr.nextRangeID,
		JobID:          jobID,
		RangeStart:     cursor,
		RangeEnd:       end,
		NextID:         cursor,
		Status:         domain.JobStatusRunning,
		LeasedBy:       workerID,
		LeaseExpiresAt: now.Add(leaseTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	jr.ranges[carved.ID] = carved
	if jr.jobRanges[jobID] == nil {
		jr.jobRanges[jobID] = make(map[int]struct{})
	}
	jr.jobRanges[jobID][carved.ID] = struct{}{}
	return carved, nil
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
		if r.LeasedBy != workerID || r.LeaseExpired(now) || r.Status != domain.JobStatusRunning {
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
