package domain

import "time"

// Range is a contiguous block of identifiers leased to a single worker
type Range struct {
	ID             int       `json:"id" form:"id" binding:"min=0"`
	JobID          int       `json:"job_id" form:"job_id" binding:"min=0,required"`
	RangeStart     int       `json:"range_start" form:"range_start" binding:"min=0,required"`
	RangeEnd       int       `json:"range_end" form:"range_end" binding:"min=0,required,gtefield=RangeStart"`
	NextID         int       `json:"next_id" form:"next_id" binding:"min=0"`
	Status         JobStatus `json:"status" form:"status" binding:"required,oneof=created validating running pausing paused stopping stopped completed failed"`
	LeasedBy       int       `json:"leased_by" form:"leased_by" binding:"min=-1"`
	LeaseExpiresAt time.Time `json:"lease_expires_at" form:"lease_expires_at"`
	Attempt        int       `json:"attempt" form:"attempt" binding:"min=0"`
	CreatedAt      time.Time `json:"created_at" form:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" form:"updated_at"`
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
