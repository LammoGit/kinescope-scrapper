package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"fmt"
)

// Errors
var (
	ErrEventStorageUnavailable = utility.NewInternalError("event storage is not configured")
)

// EventService exposes coordinator and worker events
type EventService struct {
	storage st.Storage
	hub     *sse.Hub
}

// NewEventService creates a new EventService
func NewEventService(hub *sse.Hub, storage st.Storage) (*EventService, error) {
	if storage == nil {
		return nil, ErrEventStorageUnavailable
	}
	return &EventService{storage: storage, hub: hub}, nil
}

// GetEvents returns every stored event
func (s *EventService) GetEvents(ctx context.Context) ([]domain.Event, error) {
	return s.storage.GetEvents(ctx)
}

// GetEventsByJobID returns every event belonging to a job
func (s *EventService) GetEventsByJobID(ctx context.Context, id int) ([]domain.Event, error) {
	return s.storage.GetEventsByJobID(ctx, id)
}

// GetEventsByWorkerID returns every event belonging to a worker
func (s *EventService) GetEventsByWorkerID(ctx context.Context, id int) ([]domain.Event, error) {
	return s.storage.GetEventsByWorkerID(ctx, id)
}

// AddEvent stores an event and publishes it to the matching SSE topics
func (s *EventService) AddEvent(ctx context.Context, event domain.Event) error {
	err := s.storage.AddEvent(ctx, event)

	if err == nil && s.hub != nil {
		topics := []string{"events-*"}
		if event.JobID > 0 {
			topics = append(topics, fmt.Sprintf("events-job-%d", event.JobID))
		}
		if event.WorkerID > 0 {
			topics = append(topics, fmt.Sprintf("events-worker-%d", event.WorkerID))
		}
		s.hub.PublishAll(topics, sse.SSEMessage{Event: "event", Data: event})
	}

	return err
}
