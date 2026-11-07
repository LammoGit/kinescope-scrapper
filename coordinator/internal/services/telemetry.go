package services

import (
	"coordinator/internal/domain"
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

func (s *TelemetryService) GetWorkersMetrics() []domain.Metrics {
	return nil
}

func (s *TelemetryService) GetWorkerMetricsByID(id int) ([]domain.Metrics, error) {
	return nil, nil
}

func (s *TelemetryService) GetJobsMetrics() []domain.Metrics {
	return nil
}

func (s *TelemetryService) GetJobMetricsByID(id int) ([]domain.Metrics, error) {
	return nil, nil
}
