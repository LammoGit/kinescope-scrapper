package storage_test

import (
	"bytes"
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	"crypto/ed25519"
	crand "crypto/rand"
	"errors"
	"math/rand/v2"
	"testing"
)

func randomWorker() domain.Worker {
	return domain.Worker{
		ID: rand.Int(),
		Config: domain.WorkerConfig{
			Name:        crand.Text(),
			Host:        crand.Text(),
			SSHPort:     uint16(rand.Uint()),
			SSHUser:     crand.Text(),
			SSHKey:      crand.Text(),
			Threads:     rand.Int(),
			StoragePath: crand.Text(),
		},
		Status:    crand.Text(),
		PublicKey: ed25519.PublicKey(crand.Text()),
	}
}

func equalWorkers(w1, w2 domain.Worker) bool {
	return w1.ID == w2.ID &&
		w1.Config.Name == w2.Config.Name &&
		w1.Config.Host == w2.Config.Host &&
		w1.Config.SSHPort == w2.Config.SSHPort &&
		w1.Config.SSHUser == w2.Config.SSHUser &&
		w1.Config.SSHKey == w2.Config.SSHKey &&
		w1.Config.Threads == w2.Config.Threads &&
		w1.Config.StoragePath == w2.Config.StoragePath &&
		w1.Status == w2.Status &&
		bytes.Equal(w1.PublicKey, w2.PublicKey)
}

func TestWorkers(t *testing.T) {
	t.Run("Save and get a worker", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		worker := randomWorker()
		worker.Status = ""
		worker.ID = 0

		err = s.SaveWorker(ctx, &worker)
		if err != nil {
			t.Fatalf("Failed to save a worker: %v", err)
		}

		{
			stored, err := s.GetWorkers(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || !equalWorkers(stored[0], worker) {
				t.Fatalf("Received worker isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetWorkerByID(ctx, worker.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !equalWorkers(stored, worker) {
				t.Fatalf("Received worker isn't equal to the saved one, got %v, want %v", stored, worker)
			}
		}
	})

	t.Run("GetWorkerByID request non-existant worker", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()
		_, err = s.GetWorkerByID(ctx, 0)
		if err != storage.ErrWorkerNotFoundByID {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Save, update and get a worker", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		worker := randomWorker()
		worker.ID = 0

		err = s.SaveWorker(ctx, &worker)
		if err != nil {
			t.Fatalf("Failed to save a worker: %v", err)
		}

		worker2 := worker.Clone()
		worker2.Status = worker2.Status + "123"

		err = s.SaveWorker(ctx, &worker2)
		if err != nil {
			t.Fatalf("Failed to save a worker: %v", err)
		}

		stored, err := s.GetWorkerByID(ctx, worker.ID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if equalWorkers(stored, worker) {
			t.Fatalf("Received user is equal to the previous version, got %v, want %v", stored, worker2)
		}

		if !equalWorkers(stored, worker2) {
			t.Fatalf("Received user isn't equal to either saved worker, got %v, want %v", stored, worker2)
		}
	})

	t.Run("Save, delete and get a worker", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		worker := randomWorker()
		worker.ID = 0

		err = s.SaveWorker(ctx, &worker)
		if err != nil {
			t.Fatalf("Failed to save a worker: %v", err)
		}

		err = s.DeleteWorkerByID(ctx, worker.ID)
		if err != nil {
			t.Fatalf("Failed to delete a worker: %v", err)
		}

		{
			stored, err := s.GetWorkers(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received worker isn't equal to the saved one, got %v", stored)
			}
		}

		{
			_, err := s.GetWorkerByID(ctx, worker.ID)
			if !errors.Is(err, storage.ErrWorkerNotFoundByID) {
				t.Fatalf("Unexpected error, got: %v, expected: %v", err, storage.ErrWorkerNotFoundByID)
			}
		}
	})
}
