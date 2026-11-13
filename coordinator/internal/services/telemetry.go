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
	ErrTelemetryStorageUnavailable = utility.NewInternalError("telemetry storage is not configured")
)

// TelemetryService exposes worker and job telemetry
type TelemetryService struct {
	storage st.Storage
	hub     *sse.Hub
}

// NewTelemetryService creates a new TelemetryService
func NewTelemetryService(hub *sse.Hub, storage st.Storage) (*TelemetryService, error) {
	if storage == nil {
		return nil, ErrTelemetryStorageUnavailable
	}
	return &TelemetryService{storage: storage, hub: hub}, nil
}

// GetWorkersMetrics returns the aggregated metrics of every worker
func (s *TelemetryService) GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error) {
	return s.storage.GetWorkersMetrics(ctx)
}

// GetWorkerMetricsByID returns the aggregated metrics of one worker
func (s *TelemetryService) GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	return s.storage.GetWorkerMetricsByID(ctx, id)
}

// GetJobsMetrics returns the aggregated metrics of every job
func (s *TelemetryService) GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error) {
	return s.storage.GetJobsMetrics(ctx)
}

// GetJobMetricsByID returns the aggregated metrics of one job
func (s *TelemetryService) GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	return s.storage.GetJobMetricsByID(ctx, id)
}

// AddWorkerMetricsByID stores a metrics for a worker and publishes it
func (s *TelemetryService) AddWorkerMetricsByID(ctx context.Context, id int, metrics domain.Metrics) error {
	metrics.WorkerID = id
	if err := s.storage.AddMetrics(ctx, metrics); err != nil {
		return err
	}

	if s.hub != nil {
		workerTopic := fmt.Sprintf("metrics-worker-%d", id)
		s.hub.PublishAll(
			[]string{workerTopic, "metrics-worker-*"},
			sse.SSEMessage{Event: workerTopic, Data: metrics},
		)

		if metrics.JobID > 0 {
			jobTopic := fmt.Sprintf("metrics-job-%d", metrics.JobID)
			s.hub.PublishAll(
				[]string{jobTopic, "metrics-job-*"},
				sse.SSEMessage{Event: jobTopic, Data: metrics},
			)
		}
	}

	return nil
}
