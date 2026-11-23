//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/domain"
	"sync"
)

// Won't compile if *WorkerRepository doesn't implement IWorkerRepository
var _ domain.IWorkerRepository = &WorkerRepository{}

// WorkerRepository is an in-memory worker store
type WorkerRepository struct {
	workers map[int]domain.Worker
	nextID  int
	mu      sync.RWMutex
}

// NewWorkerRepository creates a new WorkerRepository
func NewWorkerRepository() *WorkerRepository {
	return &WorkerRepository{
		workers: make(map[int]domain.Worker),
		nextID:  1,
	}
}

// SaveWorker adds new worker if id is 0 and updates otherwise.
func (wr *WorkerRepository) SaveWorker(ctx context.Context, worker *domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	// Insert
	if worker.ID == 0 {
		worker.ID = wr.nextID
		wr.nextID++

		if worker.Status == "" {
			worker.Status = domain.WorkerStatusOff
		}

		wr.workers[worker.ID] = worker.Clone()
		return nil
	}

	// Update
	if _, ok := wr.workers[worker.ID]; !ok {
		return ErrWorkerNotFoundByID
	}
	wr.workers[worker.ID] = worker.Clone()

	return nil
}

// GetWorkerByID returns a copy of a worker by its identifier
func (wr *WorkerRepository) GetWorkerByID(ctx context.Context, workerID int) (domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return domain.Worker{}, err
	}

	wr.mu.RLock()
	defer wr.mu.RUnlock()

	worker, ok := wr.workers[workerID]
	if !ok {
		return domain.Worker{}, ErrWorkerNotFoundByID
	}
	return worker, nil
}

// GetWorkers returns all registered workers ordered by identifier
func (wr *WorkerRepository) GetWorkers(ctx context.Context) ([]domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	wr.mu.RLock()
	defer wr.mu.RUnlock()

	ids := make([]int, 0, len(wr.workers))
	for id := range wr.workers {
		ids = append(ids, id)
	}

	res := make([]domain.Worker, 0, len(ids))
	for _, id := range ids {
		res = append(res, wr.workers[id])
	}
	return res, nil
}

// DeleteWorkerByID removes a worker
func (wr *WorkerRepository) DeleteWorkerByID(ctx context.Context, workerID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	if _, ok := wr.workers[workerID]; !ok {
		return ErrWorkerNotFoundByID
	}
	delete(wr.workers, workerID)
	return nil
}
