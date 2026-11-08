package repos

import (
	"coordinator/internal/domain"
	"sort"
	"sync"
)

var _ IEventRepository = &EventRepository{}

// IEventRepository stores coordinator and worker events
type IEventRepository interface {
	AddEvent(event domain.Event) domain.Event
	GetEvents() []domain.Event
	GetEventByID(id int) (domain.Event, bool)
	GetEventsByJobID(jobID int) []domain.Event
	GetEventsByWorkerID(workerID int) []domain.Event
}

// EventRepository is an in-memory event store
type EventRepository struct {
	events map[int]domain.Event
	m      sync.RWMutex
	nextID int
}

// NewEventRepository creates a new EventRepository
func NewEventRepository() *EventRepository {
	return &EventRepository{events: make(map[int]domain.Event)}
}

// AddEvent stores an event and returns it
func (er *EventRepository) AddEvent(event domain.Event) domain.Event {
	er.m.Lock()
	defer er.m.Unlock()

	er.nextID++
	er.events[er.nextID] = event
	return event
}

// GetEvents returns all stored events ordered by insertion
func (er *EventRepository) GetEvents() []domain.Event {
	er.m.RLock()
	defer er.m.RUnlock()

	ids := make([]int, 0, len(er.events))
	for id := range er.events {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	res := make([]domain.Event, 0, len(ids))
	for _, id := range ids {
		res = append(res, er.events[id])
	}
	return res
}

// GetEventByID returns a single event
func (er *EventRepository) GetEventByID(id int) (domain.Event, bool) {
	er.m.RLock()
	defer er.m.RUnlock()

	event, ok := er.events[id]
	return event, ok
}

// GetEventsByJobID returns every event belonging to a job ordered by insertion
func (er *EventRepository) GetEventsByJobID(jobID int) []domain.Event {
	er.m.RLock()
	defer er.m.RUnlock()

	return er.filter(func(event domain.Event) bool {
		return event.JobID == jobID
	})
}

// GetEventsByWorkerID returns every event belonging to a worker ordered by insertion
func (er *EventRepository) GetEventsByWorkerID(workerID int) []domain.Event {
	er.m.RLock()
	defer er.m.RUnlock()

	return er.filter(func(event domain.Event) bool {
		return event.WorkerID == workerID
	})
}

// filter returns the matching events ordered by insertion. The caller must
// hold at least a read lock.
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
