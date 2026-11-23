package storage_test

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	"math/rand/v2"
	"testing"
)

func randomMetrics() domain.Metrics {
	return domain.Metrics{
		RangeID:    rand.Int(),
		JobID:      rand.Int(),
		WorkerID:   rand.Int(),
		CPULoad:    rand.Float64(),
		MemoryLoad: rand.Float64(),
		DiskLoad:   rand.Float64(),
	}
}

func TestTelemetry(t *testing.T) {
	t.Run("Save and get one metrics batch", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		metrics := randomMetrics()

		err = s.SaveMetrics(ctx, metrics)
		if err != nil {
			t.Fatalf("Failed to save metrics: %v", err)
		}

		jtelemetry, err := s.GetJobsMetrics(ctx)
		if err != nil || len(jtelemetry) != 1 || jtelemetry[0] != metrics {
			t.Fatalf("Saved metrics aren't equal to received: %v", err)
		}

		wtelemetry, err := s.GetWorkersMetrics(ctx)
		if err != nil || len(wtelemetry) != 1 || wtelemetry[0] != metrics {
			t.Fatalf("Saved metrics aren't equal to received: %v", err)
		}

		jmetrics, err := s.GetJobMetricsByID(ctx, metrics.JobID)
		if jmetrics != metrics {
			t.Fatalf("Saved metrics aren't equal to received: %v", err)
		}

		wmetrics, err := s.GetWorkerMetricsByID(ctx, metrics.WorkerID)
		if wmetrics != metrics {
			t.Fatalf("Saved metrics aren't equal to received: %v", err)
		}

		s.Shutdown(ctx)
	})
}
