package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"sort"
	"sync"
)

// Won't compile if *EventRepository doesn't implement IEventRepository
var _ IEventRepository = &EventRepository{}

// Errors
var (
	ErrEventNotFoundByID = utility.NewNotFoundError("event with given id doesn't exist")
)

// IEventRepository stores coordinator and worker events
type IEventRepository interface {
	AddEvent(ctx context.Context, event domain.Event) error
	GetEvents(ctx context.Context) ([]domain.Event, error)
	GetEventByID(ctx context.Context, id int) (domain.Event, error)
	GetEventsByJobID(ctx context.Context, jobID int) ([]domain.Event, error)
	GetEventsByWorkerID(ctx context.Context, workerID int) ([]domain.Event, error)
}

// EventRepository is an in-memory event store
type EventRepository struct {
	events map[int]domain.Event
	nextID int
	mu     sync.RWMutex
}

// NewEventRepository creates a new EventRepository
func NewEventRepository() *EventRepository {
	return &EventRepository{
		events: make(map[int]domain.Event),
		nextID: 1,
	}
}

// AddEvent stores an event and returns it
func (er *EventRepository) AddEvent(ctx context.Context, event domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	er.nextID++
	er.events[er.nextID] = event
	return nil
}

// GetEvents returns all stored events ordered by insertion
func (er *EventRepository) GetEvents(ctx context.Context) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	ids := make([]int, 0, len(er.events))
	for id := range er.events {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	res := make([]domain.Event, 0, len(ids))
	for _, id := range ids {
		res = append(res, er.events[id])
	}
	return res, nil
}

// GetEventByID returns a single event
func (er *EventRepository) GetEventByID(ctx context.Context, id int) (domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return domain.Event{}, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	event, ok := er.events[id]
	if !ok {
		return event, ErrEventNotFoundByID
	}
	return event, nil
}

// GetEventsByJobID returns every event belonging to a job ordered by insertion
func (er *EventRepository) GetEventsByJobID(ctx context.Context, jobID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	return er.filter(func(event domain.Event) bool {
		return event.JobID == jobID
	}), nil
}

// GetEventsByWorkerID returns every event belonging to a worker ordered by insertion
func (er *EventRepository) GetEventsByWorkerID(ctx context.Context, workerID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	return er.filter(func(event domain.Event) bool {
		return event.WorkerID == workerID
	}), nil
}

// filter returns the matching events ordered by insertion
// The caller must hold at least a read lock
func (er *EventRepository) filter(match func(domain.Event) bool) []domain.Event {
	ids := make([]int, 0, len(er.events))
	for id := range er.events {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	res := make([]domain.Event, 0, len(ids))
	for _, id := range ids {
		if event := er.events[id]; match(event) {
			res = append(res, event)
		}
	}
	return res
}
