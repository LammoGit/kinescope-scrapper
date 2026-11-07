package services

import (
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
)

// TelemetryService
type TelemetryService struct {
	storage st.Storage
	hub     *sse.Hub
}

func NewTelemetryService(hub *sse.Hub, storage st.Storage) (*TelemetryService, error) {
	return &TelemetryService{storage: storage, hub: hub}, nil
}
