package services

import (
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
)

type EventService struct{}

func NewEventService(hub *sse.Hub, storage st.Storage) (*EventService, error) {
	return &EventService{}, nil
}

func (s *EventService) GetEvents() []domain.Event {
	return nil
}

func (s *EventService) GetEventsByJobID(id int) ([]domain.Event, error) {
	return nil, nil
}

func (s *EventService) GetEventsByWorkerID(id int) ([]domain.Event, error) {
	return nil, nil
}
