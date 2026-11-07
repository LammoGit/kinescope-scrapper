package services

import (
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
)

type EventService struct{}

func NewEventService(hub *sse.Hub, storage st.Storage) (*EventService, error) {
	return &EventService{}, nil
}
