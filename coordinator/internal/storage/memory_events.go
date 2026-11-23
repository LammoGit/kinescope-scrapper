//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/domain"
	"sync"
	"time"
)

// Won't compile if *EventRepository doesn't implement IEventRepository.
var _ domain.IEventRepository = &EventRepository{}

// EventRepository is an in-memory event store.
type EventRepository struct {
	events []domain.Event
	mu     sync.RWMutex
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository() *EventRepository {
	return &EventRepository{}
}

// AddEvent appends new event
func (er *EventRepository) AddEvent(ctx context.Context, event domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	er.events = append(er.events, event.Clone())

	return nil
}

// GetEvents returns all stored events ordered by insertion
func (er *EventRepository) GetEvents(ctx context.Context) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	return er.events, nil
}

// GetEventsByJobID returns every event belonging to a job ordered by insertion
func (er *EventRepository) GetEventsByJobID(ctx context.Context, jobID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	var res []domain.Event
	for _, event := range er.events {
		if event.JobID == jobID {
			res = append(res, event)
		}
	}
	return res, nil
}

// GetEventsByWorkerID returns every event belonging to a worker ordered by insertion
func (er *EventRepository) GetEventsByWorkerID(ctx context.Context, workerID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	var res []domain.Event
	for _, event := range er.events {
		if event.WorkerID == workerID {
			res = append(res, event)
		}
	}
	return res, nil
}
