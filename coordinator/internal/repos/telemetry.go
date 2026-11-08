package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"sync"
)

var _ ITelemetryRepository = &TelemetryRepository{}

// ITelemetryRepository stores and aggregates worker telemetry snapshots
type ITelemetryRepository interface {
	AddMetrics(ctx context.Context, metrics domain.Metrics) error
	GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error)
	GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error)
	GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error)
	GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error)
}

// TelemetryRepository is an in-memory telemetry store
type TelemetryRepository struct {
	metrics []domain.Metrics
	m       sync.RWMutex
}

// NewTelemetryRepository creates a new TelemetryRepository
func NewTelemetryRepository() *TelemetryRepository {
	return &TelemetryRepository{metrics: make([]domain.Metrics, 0)}
}

// AddMetrics appends a telemetry snapshot to the store
func (tr *TelemetryRepository) AddMetrics(ctx context.Context, metrics domain.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tr.m.Lock()
	defer tr.m.Unlock()
	tr.metrics = append(tr.metrics, metrics)
	return nil
}

// snapshot returns a copy of all stored snapshots
func (tr *TelemetryRepository) snapshot() []domain.Metrics {
	tr.m.RLock()
	defer tr.m.RUnlock()

	res := make([]domain.Metrics, len(tr.metrics))
	copy(res, tr.metrics)
	return res
}

// GetWorkersMetrics returns one aggregated snapshot per worker
func (tr *TelemetryRepository) GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	byWorker := make(map[int][]domain.Metrics)
	var order []int
	for _, m := range tr.snapshot() {
		if _, ok := byWorker[m.WorkerID]; !ok {
			order = append(order, m.WorkerID)
		}
		byWorker[m.WorkerID] = append(byWorker[m.WorkerID], m)
	}

	res := make([]domain.Metrics, 0, len(order))
	for _, id := range order {
		res = append(res, domain.AverageMetrics(byWorker[id]...))
	}
	return res, nil
}

// GetJobsMetrics returns one aggregated snapshot per job
func (tr *TelemetryRepository) GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	byJob := make(map[int][]domain.Metrics)
	var order []int
	for _, m := range tr.snapshot() {
		if _, ok := byJob[m.JobID]; !ok {
			order = append(order, m.JobID)
		}
		byJob[m.JobID] = append(byJob[m.JobID], m)
	}

	res := make([]domain.Metrics, 0, len(order))
	for _, id := range order {
		res = append(res, domain.AverageMetrics(byJob[id]...))
	}
	return res, nil
}

// GetWorkerMetricsByID returns the aggregated metrics of one worker
func (tr *TelemetryRepository) GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	var metrics []domain.Metrics
	for _, m := range tr.snapshot() {
		if m.WorkerID == id {
			metrics = append(metrics, m)
		}
	}
	if len(metrics) == 0 {
		return domain.Metrics{}, utility.NewNotFoundError("no metrics found for the given worker")
	}
	return domain.AverageMetrics(metrics...), nil
}

// GetJobMetricsByID returns the aggregated metrics of one job
func (tr *TelemetryRepository) GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	var metrics []domain.Metrics
	for _, m := range tr.snapshot() {
		if m.JobID == id {
			metrics = append(metrics, m)
		}
	}
	if len(metrics) == 0 {
		return domain.Metrics{}, utility.NewNotFoundError("no metrics found for the given job")
	}
	return domain.AverageMetrics(metrics...), nil
}
