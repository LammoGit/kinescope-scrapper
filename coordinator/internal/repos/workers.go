package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"sync"
)

// Won't compile if *WorkerRepository doesn't implement IWorkerRepository
var _ IWorkerRepository = &WorkerRepository{}

// Errors
var (
	ErrWorkerNotFoundByID = utility.NewNotFoundError("worker with given id doesn't exist")
)

// IWorkerRepository stores registered workers
type IWorkerRepository interface {
	GetWorkerByID(ctx context.Context, workerID int) (domain.Worker, error)
	GetWorkers(ctx context.Context) ([]domain.Worker, error)
	AddWorker(ctx context.Context, worker domain.Worker) error
	ReplaceWorker(ctx context.Context, worker domain.Worker) error
	DeleteWorkerByID(ctx context.Context, workerID int) error
}

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

// AddWorker stores a worker, assigning it an identifier when needed
func (wr *WorkerRepository) AddWorker(ctx context.Context, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	if worker.ID == 0 {
		wr.nextID++
		worker.ID = wr.nextID
	} else if worker.ID > wr.nextID {
		wr.nextID = worker.ID
	}
	if worker.Status == "" {
		worker.Status = domain.WorkerStatusOff
	}
	wr.workers[worker.ID] = worker
	return nil
}

// ReplaceWorker replaces a stored worker
func (wr *WorkerRepository) ReplaceWorker(ctx context.Context, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	if _, ok := wr.workers[worker.ID]; !ok {
		return ErrWorkerNotFoundByID
	}
	wr.workers[worker.ID] = worker
	return nil
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
