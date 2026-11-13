package domain

import (
	"slices"
	"time"
)

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

// Job is a scraping task and its current state
type Job struct {
	ID         int       `json:"id" form:"id" binding:"omitempty,min=1"`
	Config     JobConfig `json:"config" form:"config" binding:"required"`
	Workers    []int     `json:"workers" form:"workers" binding:"omitempty"`
	Status     string    `json:"status" form:"status" binding:"required,oneof=created validating running pausing paused stopping stopped completed failed"`
	Paused     bool      `json:"paused" form:"paused" binding:"omitempty"`
	Expiration time.Time `json:"expiration" form:"expiration" binding:"omitempty"`
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

// JobRange is the inclusive range of identifiers a job processes
type JobRange struct {
	From int `json:"from" form:"from" binding:"omitempty,min=0"`
	To   int `json:"to" form:"to" binding:"omitempty,gtefield=From"`
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

// Leased reports whether the range is currently assigned to a worker
func (r *Range) Leased() bool {
	return r.LeasedBy != -1
}

// LeaseExpired reports whether the range's lease is no longer valid
func (r *Range) LeaseExpired(now time.Time) bool {
	return r.Leased() && !r.LeaseExpiresAt.IsZero() && !now.Before(r.LeaseExpiresAt)
}

// Exhausted reports whether every identifier of the range was processed
func (r *Range) Exhausted() bool {
	return r.NextID > r.RangeEnd
}
