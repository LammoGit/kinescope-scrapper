package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ IEventRepository = &EventRepository{}

type IEventRepository interface {
	GetEvents() []domain.Event
	GetEventByID(id int) (domain.Event, bool)
	GetEventsByJobID(jobId int) []domain.Event
	GetEventsByWorkerID(workerId int) []domain.Event
}

type EventRepository struct {
	events map[int]domain.Event
	m      sync.RWMutex
}

func (er *EventRepository) GetEvents() []domain.Event                       {}
func (er *EventRepository) GetEventByID(id int) (domain.Event, bool)        {}
func (er *EventRepository) GetEventsByJobID(jobId int) []domain.Event       {}
func (er *EventRepository) GetEventsByWorkerID(workerId int) []domain.Event {}

