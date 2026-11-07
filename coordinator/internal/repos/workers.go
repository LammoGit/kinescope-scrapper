package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ IWorkerRepository = &WorkerRepository{}

type IWorkerRepository interface {
	GetWorkerByID(workerId int) (*domain.Worker, bool)
}

type WorkerRepository struct {
	workers map[int]domain.Worker
	m       sync.RWMutex
}

func (wr *WorkerRepository) GetWorkerByID(workerId int) (*domain.Worker, bool) {
	wr.m.RLock()
	defer wr.m.RUnlock()
	worker, ok := wr.workers[workerId]
	return &worker, ok
}

func (wr *WorkerRepository) GetWorkers() {}

func (wr *WorkerRepository) GetWorkerInfo(workerID string) (ServerInfo, bool) {}

func (wr *WorkerRepository) GetWorkerMetrics(workerID string) (ServerMetrics, bool) {}

func (wr *WorkerRepository) UpdateServerMetrics(workerID string, metrics ServerMetrics) error {}

func (wr *WorkerRepository) UpdateServerInfo(workerID string, info ServerInfo) {}

func (wr *WorkerRepository) RegisterServer(info ServerInfo) (workerID, sshKey string, err error) {}
