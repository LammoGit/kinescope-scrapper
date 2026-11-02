package domain

import "time"

type Range struct {
	ID             int       `json:"id"`
	JobID          int       `json:"job_id"`
	RangeStart     int       `json:"range_start"`
	RangeEnd       int       `json:"range_end"`
	NextID         int       `json:"next_id"`
	Status         JobStatus `json:"status"`
	LeasedBy       int       `json:"leased_by"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
	Attempt        int       `json:"attempt"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
