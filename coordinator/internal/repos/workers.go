package repos

import (
	"coordinator/internal/domain"
	"sort"
	"sync"
)

var _ IWorkerRepository = &WorkerRepository{}

// IWorkerRepository stores registered workers
type IWorkerRepository interface {
	GetWorkerByID(workerID int) (*domain.Worker, bool)
	GetWorkers() []domain.Worker
	AddWorker(worker domain.Worker) domain.Worker
	UpdateWorker(worker domain.Worker) bool
	DeleteWorkerByID(workerID int) bool
}

// WorkerRepository is an in-memory worker store
type WorkerRepository struct {
	workers map[int]domain.Worker
	m       sync.RWMutex
	nextID  int
}

// NewWorkerRepository creates a new WorkerRepository
func NewWorkerRepository() *WorkerRepository {
	return &WorkerRepository{workers: make(map[int]domain.Worker)}
}

// GetWorkerByID returns a copy of a worker by its identifier
func (wr *WorkerRepository) GetWorkerByID(workerID int) (*domain.Worker, bool) {
	wr.m.RLock()
	defer wr.m.RUnlock()

	worker, ok := wr.workers[workerID]
	if !ok {
		return nil, false
	}
	return &worker, true
}

// GetWorkers returns all registered workers ordered by identifier
func (wr *WorkerRepository) GetWorkers() []domain.Worker {
	wr.m.RLock()
	defer wr.m.RUnlock()

	ids := make([]int, 0, len(wr.workers))
	for id := range wr.workers {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	res := make([]domain.Worker, 0, len(ids))
	for _, id := range ids {
		res = append(res, wr.workers[id])
	}
	return res
}

// AddWorker stores a worker, assigning it an identifier when needed
func (wr *WorkerRepository) AddWorker(worker domain.Worker) domain.Worker {
	wr.m.Lock()
	defer wr.m.Unlock()

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
	return worker
}

// UpdateWorker replaces a stored worker
func (wr *WorkerRepository) UpdateWorker(worker domain.Worker) bool {
	wr.m.Lock()
	defer wr.m.Unlock()

	if _, ok := wr.workers[worker.ID]; !ok {
		return false
	}
	wr.workers[worker.ID] = worker
	return true
}

// DeleteWorkerByID removes a worker
func (wr *WorkerRepository) DeleteWorkerByID(workerID int) bool {
	wr.m.Lock()
	defer wr.m.Unlock()

	if _, ok := wr.workers[workerID]; !ok {
		return false
	}
	delete(wr.workers, workerID)
	return true
}
