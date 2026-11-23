//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/domain"
	"sync"
)

// Won't compile if *TelemetryRepository doesn't implement ITelemetryRepository
var _ domain.ITelemetryRepository = &TelemetryRepository{}

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

// SaveMetrics adds metrics if telemetry key is unique, updates otherwise.
func (tr *TelemetryRepository) SaveMetrics(ctx context.Context, metrics domain.Metrics) error {
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
	tr.metrics[key] = metrics.Clone()

	if jk, ok := tr.jobKeys[metrics.JobID]; ok {
		jk[key] = struct{}{}
	} else {
		tr.jobKeys[metrics.JobID] = map[TelemetryKey]struct{}{key: {}}
	}

	if wk, ok := tr.workerKeys[metrics.WorkerID]; ok {
		wk[key] = struct{}{}
	} else {
		tr.workerKeys[metrics.WorkerID] = map[TelemetryKey]struct{}{key: {}}
	}

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
	for _, v := range tr.metrics {
		res = append(res, v.Clone())
	}
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
			res = append(res, tr.metrics[key].Clone())
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
