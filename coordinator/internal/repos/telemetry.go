package repos

import (
	"context"
	"coordinator/internal/domain"
	"maps"
	"slices"
	"sync"
)

// Won't compile if *TelemetryRepository doesn't implement ITelemetryRepository
var _ ITelemetryRepository = &TelemetryRepository{}

// ITelemetryRepository stores and aggregates worker telemetry snapshots
type ITelemetryRepository interface {
	AddMetrics(ctx context.Context, metrics domain.Metrics) error
	GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error)
	GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error)
	GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error)
	GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error)
}

// TelemetryKey unique key of metrics
type TelemetryKey struct {
	JobID, RangeID, WorkerID int
}

// TelemetryRepository is an in-memory telemetry store
type TelemetryRepository struct {
	metrics    map[TelemetryKey]domain.Metrics
	jobKeys    map[int]map[TelemetryKey]struct{}
	workerKeys map[int]map[TelemetryKey]struct{}
	mu         sync.RWMutex
}

// NewTelemetryRepository creates a new TelemetryRepository
func NewTelemetryRepository() *TelemetryRepository {
	return &TelemetryRepository{
		metrics:    make(map[TelemetryKey]domain.Metrics),
		jobKeys:    make(map[int]map[TelemetryKey]struct{}),
		workerKeys: make(map[int]map[TelemetryKey]struct{}),
	}
}

// AddMetrics appends metrics
func (tr *TelemetryRepository) AddMetrics(ctx context.Context, metrics domain.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()
	key := TelemetryKey{
		JobID:    metrics.JobID,
		RangeID:  metrics.RangeID,
		WorkerID: metrics.WorkerID,
	}
	tr.metrics[key] = metrics
	tr.jobKeys[metrics.JobID][key] = struct{}{}
	tr.workerKeys[metrics.WorkerID][key] = struct{}{}

	return nil
}

// GetWorkersMetrics returns one aggregated metrics per worker
func (tr *TelemetryRepository) GetWorkersMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var res []domain.Metrics
	res = slices.AppendSeq(res, maps.Values(tr.metrics))
	return res, nil
}

// GetJobsMetrics returns one aggregated metrics per job
func (tr *TelemetryRepository) GetJobsMetrics(ctx context.Context) ([]domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var res []domain.Metrics
	for _, keys := range tr.jobKeys {
		for key := range keys {
			res = append(res, tr.metrics[key])
		}
	}
	return res, nil
}

// GetWorkerMetricsByID returns aggregated metrics of one worker
func (tr *TelemetryRepository) GetWorkerMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	var metrics []domain.Metrics
	keys, ok := tr.workerKeys[id]
	if !ok {
		return domain.Metrics{}, ErrWorkerNotFoundByID
	}
	for key := range keys {
		metrics = append(metrics, tr.metrics[key])
	}
	return domain.AverageMetrics(metrics...), nil
}

// GetJobMetricsByID returns aggregated metrics of one job
func (tr *TelemetryRepository) GetJobMetricsByID(ctx context.Context, id int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	var metrics []domain.Metrics
	keys, ok := tr.jobKeys[id]
	if !ok {
		return domain.Metrics{}, ErrJobNotFoundByID
	}
	for key := range keys {
		metrics = append(metrics, tr.metrics[key])
	}
	return domain.AverageMetrics(metrics...), nil
}

// GetMetricsByWorkerJobID returns aggregated metrics of one job and one worker
func (tr *TelemetryRepository) GetMetricsByWorkerJobID(ctx context.Context, workerID, jobID int) (domain.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return domain.Metrics{}, err
	}

	var metrics []domain.Metrics
	keys, ok := tr.jobKeys[jobID]
	if !ok {
		return domain.Metrics{}, ErrJobNotFoundByID
	}
	for key := range keys {
		temp := tr.metrics[key]
		if temp.WorkerID != workerID {
			continue
		}
		metrics = append(metrics, temp)
	}
	return domain.AverageMetrics(metrics...), nil
}
