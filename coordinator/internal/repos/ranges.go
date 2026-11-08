package repos

import (
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"sort"
	"sync"
	"time"
)

// rangeLeaseTTL is how long a range stays claimed by a worker without a heartbeat
const rangeLeaseTTL = 2 * time.Minute

var _ IRangeRepository = &RangeRepository{}

// IRangeRepository stores ranges and manages their leases
type IRangeRepository interface {
	AddRange(rangeBlock domain.Range) domain.Range
	AddRanges(jobID, from, to int) []domain.Range
	GetRangeByID(id int) (domain.Range, bool)
	GetRangesByJobID(jobID int) []domain.Range
	GetUnassignedRanges() ([]domain.Range, error)
	TryClaimRange(rangeID, workerID int) bool
	RenewLeases(workerID int) int
	ReleaseRange(rangeID int) error
	CompleteRange(rangeID int) error
	DeleteRangesByJobID(jobID int)
}

// RangeRepository is an in-memory range store
type RangeRepository struct {
	ranges map[int]domain.Range
	m      sync.RWMutex
	nextID int
}

// NewRangeRepository creates a new RangeRepository
func NewRangeRepository() *RangeRepository {
	return &RangeRepository{ranges: make(map[int]domain.Range)}
}

// AddRange stores a range, assigning it an identifier when needed
func (rr *RangeRepository) AddRange(rangeBlock domain.Range) domain.Range {
	rr.m.Lock()
	defer rr.m.Unlock()

	if rangeBlock.ID == 0 {
		rr.nextID++
		rangeBlock.ID = rr.nextID
	} else if rangeBlock.ID > rr.nextID {
		rr.nextID = rangeBlock.ID
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

	rr.ranges[rangeBlock.ID] = rangeBlock
	return rangeBlock
}

// AddRanges splits [from, to] into blocks and stores them
func (rr *RangeRepository) AddRanges(jobID, from, to int) []domain.Range {
	if to < from {
		return nil
	}

	const blockSize = 1000
	var res []domain.Range
	for start := from; start <= to; start += blockSize {
		end := start + blockSize - 1
		if end > to {
			end = to
		}
		res = append(res, rr.AddRange(domain.Range{
			JobID:      jobID,
			RangeStart: start,
			RangeEnd:   end,
			NextID:     start,
			Status:     domain.JobStatusCreated,
		}))
	}
	return res
}

// GetRangeByID returns a copy of a range by its identifier
func (rr *RangeRepository) GetRangeByID(id int) (domain.Range, bool) {
	rr.m.RLock()
	defer rr.m.RUnlock()

	r, ok := rr.ranges[id]
	return r, ok
}

// GetRangesByJobID returns all ranges belonging to a job ordered by start
func (rr *RangeRepository) GetRangesByJobID(jobID int) []domain.Range {
	rr.m.RLock()
	defer rr.m.RUnlock()

	var res []domain.Range
	for _, r := range rr.ranges {
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
	return res
}

// GetUnassignedRanges returns ranges that are ready to be leased
func (rr *RangeRepository) GetUnassignedRanges() ([]domain.Range, error) {
	rr.m.RLock()
	defer rr.m.RUnlock()

	now := time.Now()
	var res []domain.Range
	for _, r := range rr.ranges {
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
func (rr *RangeRepository) TryClaimRange(rangeID, workerID int) bool {
	rr.m.Lock()
	defer rr.m.Unlock()

	r, ok := rr.ranges[rangeID]
	if !ok {
		return false
	}
	if r.Status == domain.JobStatusCompleted || r.Status == domain.JobStatusStopped || r.Status == domain.JobStatusFailed {
		return false
	}

	now := time.Now()
	if r.Leased() && !r.LeaseExpired(now) {
		return false
	}

	r.LeasedBy = workerID
	r.LeaseExpiresAt = now.Add(rangeLeaseTTL)
	r.Status = domain.JobStatusRunning
	r.UpdatedAt = now
	rr.ranges[r.ID] = r
	return true
}

// RenewLeases extends every active lease of a worker
func (rr *RangeRepository) RenewLeases(workerID int) int {
	rr.m.Lock()
	defer rr.m.Unlock()

	now := time.Now()
	renewed := 0
	for id, r := range rr.ranges {
		if r.LeasedBy != workerID || r.Status != domain.JobStatusRunning {
			continue
		}
		r.LeaseExpiresAt = now.Add(rangeLeaseTTL)
		r.UpdatedAt = now
		rr.ranges[id] = r
		renewed++
	}
	return renewed
}

// ReleaseRange unassigns a range from its worker while keeping its progress
func (rr *RangeRepository) ReleaseRange(rangeID int) error {
	rr.m.Lock()
	defer rr.m.Unlock()

	r, ok := rr.ranges[rangeID]
	if !ok {
		return utility.NewNotFoundError("range with the given id doesn't exist")
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
	rr.ranges[r.ID] = r
	return nil
}

// CompleteRange marks a range as fully processed
func (rr *RangeRepository) CompleteRange(rangeID int) error {
	rr.m.Lock()
	defer rr.m.Unlock()

	r, ok := rr.ranges[rangeID]
	if !ok {
		return utility.NewNotFoundError("range with the given id doesn't exist")
	}

	r.LeasedBy = 0
	r.LeaseExpiresAt = time.Time{}
	r.Status = domain.JobStatusCompleted
	r.UpdatedAt = time.Now()
	rr.ranges[r.ID] = r
	return nil
}

// DeleteRangesByJobID removes every range of a job
func (rr *RangeRepository) DeleteRangesByJobID(jobID int) {
	rr.m.Lock()
	defer rr.m.Unlock()

	for id, r := range rr.ranges {
		if r.JobID == jobID {
			delete(rr.ranges, id)
		}
	}
}
