package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ ITelemetryRepository = &TelemetryRepository{}

type ITelemetryRepository interface {
	GetWorkersMetrics() []domain.Metrics
	GetJobsMetrics() []domain.Metrics
	GetWorkerMetricsByID(id int) (domain.Metrics, bool)
	GetJobMetricsByID(id int) (domain.Metrics, bool)
}

type TelemetryRepository struct {
	metrics []domain.Metrics
	m       sync.RWMutex
}

func (tr *TelemetryRepository) GetWorkerMetricsByID(id int) (domain.Metrics, bool) {
	tr.m.RLock()
	defer tr.m.RUnlock()

	var metrics []domain.Metrics
	for _, m := range tr.metrics {
		if m.WorkerID == id {
			metrics = append(metrics, m)
		}
	}
	return domain.AverageMetrics(metrics), len(metrics) > 0
}

func (tr *TelemetryRepository) GetJobMetricsByID(id int) (domain.Metrics, bool) {
	tr.m.RLock()
	defer tr.m.RUnlock()

	var metrics []domain.Metrics
	for _, m := range tr.metrics {
		if m.JobID == id {
			metrics = append(metrics, m)
		}
	}
	return domain.AverageMetrics(metrics), len(metrics) > 0
}
