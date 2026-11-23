package domain

import (
	"cmp"
	"context"
	"coordinator/internal/utility"
	"slices"
	"time"
)

// Errors
var (
	ErrRangeIsTaken            = utility.NewConflictError("range is already taken")
	ErrRangeIsOutsideOfJob     = utility.NewConflictError("range is outside of the job's bounds")
	ErrRangeIsCompleted        = utility.NewConflictError("range is already completed")
	ErrJobIsCompleted          = utility.NewConflictError("job is already completed")
	ErrWorkerDoesNotLeaseRange = utility.NewConflictError("range isn't leased by the worker with the given id")
)

// IJobRepository stores scraping jobs and their state
type IJobRepository interface {
	// Jobs
	SaveJob(ctx context.Context, job *Job) error
	GetJobByID(ctx context.Context, id int) (Job, error)
	GetJobs(ctx context.Context) ([]Job, error)
	DeleteJobByID(ctx context.Context, id int) error

	// Ranges
	SaveRange(ctx context.Context, rangeBlock *Range) error
	GetRangeByID(ctx context.Context, id int) (Range, error)
	GetRangesByJobID(ctx context.Context, jobID int) ([]Range, error)
	GetRangesByWorkerID(ctx context.Context, workerID int) ([]Range, error)
	DeleteRangesByJobID(ctx context.Context, jobID int) error
	RenewLeases(ctx context.Context, workerID int, ttl time.Duration) error
}

// Job statuses
const (
	JobStatusCreated    string = "created"
	JobStatusValidating string = "validating"
	JobStatusRunning    string = "running"
	JobStatusPausing    string = "pausing"
	JobStatusPaused     string = "paused"
	JobStatusStopping   string = "stopping"
	JobStatusStopped    string = "stopped"
	JobStatusCompleted  string = "completed"
	JobStatusFailed     string = "failed"
)

// JobRange is the inclusive range of identifiers a job processes, a zero To leaves the upper bound open
type JobRange struct {
	From int `json:"from" form:"from" binding:"omitempty,min=0"`
	To   int `json:"to" form:"to" binding:"omitempty,gtefield=From"`
}

// JobConfig contains configuration of a job
type JobConfig struct {
	Name                     string         `json:"name" form:"name" binding:"required"`
	Range                    JobRange       `json:"range" form:"range" binding:"required"`
	RequestTimeout           int            `json:"request_timeout_seconds" form:"request_timeout_seconds" binding:"omitempty,gt=0"`
	MaxTries                 int            `json:"max_tries" form:"max_retries" binding:"omitempty,min=0"`
	GlobalRateLimitRPS       int            `json:"global_rate_limit_rps" form:"global_rate_limit_rps" binding:"omitempty,min=0"`
	StopOnConsecutiveMissing int            `json:"stop_on_consecutive_missing" form:"stop_on_consecutive_missing" binding:"omitempty,min=0"`
	Servers                  []WorkerConfig `json:"servers" form:"servers" binding:"omitempty"`
}

// Range is a contiguous block of identifiers leased to a single worker
type Range struct {
	ID             int       `json:"id" form:"id" binding:"required,min=1"`
	JobID          int       `json:"job_id" form:"job_id" binding:"required,min=1"`
	RangeStart     int       `json:"range_start" form:"range_start" binding:"required,min=0"`
	RangeEnd       int       `json:"range_end" form:"range_end" binding:"required,gtefield=RangeStart"`
	NextID         int       `json:"next_id" form:"next_id" binding:"omitempty,gtefield=RangeStart"`
	Status         string    `json:"status" form:"status" binding:"required,oneof=created validating running pausing paused stopping stopped completed failed"`
	LeasedBy       int       `json:"leased_by" form:"leased_by" binding:"omitempty,min=0"`
	LeaseExpiresAt time.Time `json:"lease_expires_at" form:"lease_expires_at" binding:"omitempty"`
	Attempt        int       `json:"attempt" form:"attempt" binding:"omitempty,min=0"`
	CreatedAt      time.Time `json:"created_at" form:"created_at" binding:"required"`
	UpdatedAt      time.Time `json:"updated_at" form:"updated_at" binding:"required"`
}

// Job is a scraping task and its current state
type Job struct {
	ID         int       `json:"id" form:"id" binding:"omitempty,min=1"`
	Config     JobConfig `json:"config" form:"config" binding:"required"`
	Workers    []int     `json:"workers" form:"workers" binding:"omitempty"`
	Ranges     []Range   `json:"ranges" form:"ranges" binding:"omitempty"`
	Status     string    `json:"status" form:"status" binding:"required,oneof=created validating running pausing paused stopping stopped completed failed"`
	Paused     bool      `json:"paused" form:"paused" binding:"omitempty"`
	Expiration time.Time `json:"expiration" form:"expiration" binding:"omitempty"`
}

// Clone creates a deep-copy of the calling JobConfig
func (jc JobConfig) Clone() JobConfig {
	clone := jc
	clone.Servers = slices.Clone(jc.Servers)
	return clone
}

// Clone creates a deep-copy of the calling Range
func (r Range) Clone() Range {
	return r
}

// Leased reports whether the range is currently assigned to a worker
func (r Range) Leased() bool {
	return r.LeasedBy != 0
}

// LeaseExpired reports whether the range's lease is no longer valid
func (r Range) LeaseExpired(now time.Time) bool {
	return !r.Leased() || (!r.LeaseExpiresAt.IsZero() && !now.Before(r.LeaseExpiresAt))
}

// Exhausted reports whether every identifier of the range was processed
func (r Range) Exhausted() bool {
	return r.NextID > r.RangeEnd
}

// Lease assigns worker to the range
func (r *Range) Lease(workerID int, ttl time.Duration) error {
	now := time.Now()

	if r.Status == JobStatusCompleted {
		return ErrJobIsCompleted
	}

	if r.Exhausted() {
		return ErrRangeIsCompleted
	}

	if !r.LeaseExpired(now) && r.LeasedBy != workerID {
		return ErrRangeIsTaken
	}

	r.LeasedBy = workerID
	r.LeaseExpiresAt = now.Add(ttl)
	r.Status = JobStatusRunning
	r.Attempt++
	r.UpdatedAt = now
	return nil
}

// Renew extends active lease of a worker for the range
func (r *Range) Renew(workerID int, ttl time.Duration) error {
	now := time.Now()
	if r.LeasedBy != workerID || r.LeaseExpired(now) {
		return ErrWorkerDoesNotLeaseRange
	}
	r.LeaseExpiresAt = now.Add(ttl)
	r.UpdatedAt = now
	return nil
}

// Release unassigns current worker from the range
func (r *Range) Release() {
	r.LeasedBy = 0
	r.LeaseExpiresAt = time.Time{}
	r.UpdatedAt = time.Now()
	if r.Exhausted() {
		r.Status = JobStatusCompleted
	} else {
		r.Status = JobStatusCreated
	}
}

// Complete marks a range as fully processed
func (r *Range) Complete() {
	r.LeasedBy = 0
	r.LeaseExpiresAt = time.Time{}
	r.Status = JobStatusCompleted
	r.UpdatedAt = time.Now()
}

// Clone creates a deep-copy of the calling Job
func (j Job) Clone() Job {
	clone := j
	clone.Config = j.Config.Clone()
	clone.Workers = slices.Clone(j.Workers)
	clone.Ranges = slices.Clone(j.Ranges)
	return clone
}

// rangeTaken checks whether range collides with other ranges of the job
func (j Job) rangeTaken(from, to int) bool {
	for _, r := range j.Ranges {
		if r.RangeStart <= from && from <= r.RangeEnd {
			return true
		}
		if r.RangeStart <= to && to <= r.RangeEnd {
			return true
		}
	}
	return false
}

// AssignWorkers assignes workers passed as variadic arguments
func (j *Job) AssignWorkers(ws ...int) {
	j.DeassignWorkers(ws...)
	j.Workers = append(j.Workers, ws...)
}

// DeassignWorkers deassignes workers passed as variadic arguments
func (j *Job) DeassignWorkers(ws ...int) {
	if len(ws) == 0 {
		j.Workers = []int{}
		return
	}

	wss := make(map[int]struct{})
	for _, w := range ws {
		wss[w] = struct{}{}
	}

	j.Workers = slices.DeleteFunc(j.Workers, func(w int) bool {
		_, ok := wss[w]
		return ok
	})
}

// Start moves a job into the running state
func (j *Job) Start() {
	j.Status = JobStatusRunning
	j.Paused = false
}

// Pause moves a job into the paused state
func (j *Job) Pause() {
	j.Status = JobStatusPaused
	j.Paused = true
}

// Resume moves a paused job back into the running state
func (j *Job) Resume() {
	j.Status = JobStatusRunning
	j.Paused = false
}

// Stop moves a job into the stopped state
func (j *Job) Stop() {
	j.Status = JobStatusStopped
	j.Paused = true
	j.DeassignWorkers()
	j.Ranges = nil
}

// AddRange adds new range to id
func (j *Job) AddRange(r Range) error {
	if r.RangeStart < j.Config.Range.From || r.RangeEnd > j.Config.Range.To {
		return ErrRangeIsOutsideOfJob
	}
	if j.rangeTaken(r.RangeStart, r.RangeEnd) {
		return ErrRangeIsTaken
	}
	j.Ranges = append(j.Ranges, r.Clone())
	return nil
}

// DeleteRange deletes range from the Job by range's id
func (j *Job) DeleteRange(rid int) {
	j.Ranges = slices.DeleteFunc(j.Ranges, func(r Range) bool {
		return r.ID == rid
	})
}

// DeleteRanges deletes all ranges in ids aren't specified,
// deletes only specified otherwise
func (j *Job) DeleteRanges(rids ...int) {
	if len(rids) == 0 {
		j.Ranges = nil
		return
	}

	set := make(map[int]struct{})
	for _, rid := range rids {
		set[rid] = struct{}{}
	}

	j.Ranges = slices.DeleteFunc(j.Ranges, func(r Range) bool {
		_, ok := set[r.ID]
		return ok
	})
}

// ClaimNextRange attempts to lease a range to a worker
func (j *Job) ClaimNextRange(workerID int, leaseTTL time.Duration) (Range, error) {
	if j.Status == JobStatusCompleted {
		return Range{}, ErrJobIsCompleted
	}

	now := time.Now()

	slices.SortFunc(j.Ranges, func(a, b Range) int {
		if a.RangeStart == b.RangeStart {
			return cmp.Compare(a.ID, b.ID)
		}
		return cmp.Compare(a.RangeStart, b.RangeStart)
	})

	for i := range j.Ranges {
		r := &j.Ranges[i]
		if r.Status == JobStatusCompleted || r.Exhausted() {
			continue
		}
		if !r.LeaseExpired(now) {
			continue
		}
		r.Lease(workerID, leaseTTL)
		return r.Clone(), nil
	}

	cursor := j.Config.Range.From
	gapEnd := j.Config.Range.To
	maxSize := 1000
	open := j.Config.Range.To <= 0
	tail := true

	for _, r := range j.Ranges {
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
		gapEnd = cursor + maxSize - 1
	} else if !open && cursor > j.Config.Range.To {
		return Range{}, ErrRangeIsTaken
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

	carved := Range{
		JobID:          j.ID,
		RangeStart:     cursor,
		RangeEnd:       cursor + size - 1,
		NextID:         cursor,
		Status:         JobStatusRunning,
		LeasedBy:       workerID,
		LeaseExpiresAt: now.Add(leaseTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	j.Ranges = append(j.Ranges, carved)
	return carved.Clone(), nil
}
