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
func NewTelemetryService(_ context.Context, hub *sse.Hub, storage st.Storage) (*TelemetryService, error) {
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
	if err := s.storage.SaveMetrics(ctx, metrics); err != nil {
		return err
	}

	if s.hub != nil {
		// Publish to the topics that the /stream routes subscribe to, using the event names that the frontend listens for
		s.hub.PublishAll(
			[]string{
				fmt.Sprintf("worker-metrics-%d", id),
				"worker-metrics-*",
			},
			sse.SSEMessage{Event: "worker.metrics", Data: metrics},
		)

		if metrics.JobID > 0 {
			s.hub.PublishAll(
				[]string{
					fmt.Sprintf("job-metrics-%d", metrics.JobID),
					"job-metrics-*",
				},
				sse.SSEMessage{Event: "job.metrics", Data: metrics},
			)
		}
	}

	return nil
}
