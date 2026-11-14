package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"crypto/ed25519"
	"fmt"
)

// Errors
var (
	ErrWorkerStorageUnavailable = utility.NewInternalError("worker storage is not configured")
	ErrWorkerKeyIsEmpty         = utility.NewInternalError("worker doesn't have a signature key")
)

// WorkerService exposes registered workers
type WorkerService struct {
	storage st.Storage
	hub     *sse.Hub
}

// NewWorkerService creates a new WorkerService
func NewWorkerService(hub *sse.Hub, storage st.Storage) (*WorkerService, error) {
	if storage == nil {
		return nil, ErrWorkerStorageUnavailable
	}
	return &WorkerService{storage: storage, hub: hub}, nil
}

// CreateWorker registers a new worker and returns it
func (s *WorkerService) CreateWorker(ctx context.Context, host string, port int, user string, pubKey ed25519.PublicKey, threads int, storagePath string, minFreeSpaceGB int) (domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return domain.Worker{}, err
	}

	worker := domain.Worker{
		Config: domain.WorkerConfig{
			Name:           "worker",
			Host:           host,
			SSHPort:        port,
			SSHUser:        user,
			Threads:        threads,
			StoragePath:    storagePath,
			MinFreeSpaceGB: minFreeSpaceGB,
		},
		Status:    domain.WorkerStatusOff,
		PublicKey: pubKey,
	}

	err := s.storage.AddWorker(ctx, worker)
	if err != nil {
		return worker, err
	}
	worker.Config.Name = fmt.Sprintf("worker-%d", worker.ID)
	s.storage.ReplaceWorker(ctx, worker)

	s.publish(worker)
	return worker, nil
}

// GetWorkerSignatureKeyByID returns the ed25519 key used to verify a worker's request signatures
func (s *WorkerService) GetWorkerSignatureKeyByID(ctx context.Context, id int) (ed25519.PublicKey, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	worker, err := s.storage.GetWorkerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(worker.PublicKey) == 0 {
		return nil, ErrWorkerKeyIsEmpty
	}
	return worker.PublicKey, nil
}

// GetWorkers returns every registered worker
func (s *WorkerService) GetWorkers(ctx context.Context) ([]domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.storage.GetWorkers(ctx)
}

// GetWorkerByID returns a worker with the given id
func (s *WorkerService) GetWorkerByID(ctx context.Context, id int) (domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return domain.Worker{}, err
	}

	worker, err := s.storage.GetWorkerByID(ctx, id)
	if err != nil {
		return worker, err
	}
	return worker, nil
}

// ReplaceWorkerByID replaces current worker data with new one
func (s *WorkerService) ReplaceWorkerByID(ctx context.Context, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := s.storage.ReplaceWorker(ctx, worker)
	if err != nil {
		return err
	}
	s.publish(worker)

	return nil
}

// DeleteWorkerByID removes a worker
func (s *WorkerService) DeleteWorkerByID(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := s.storage.DeleteWorkerByID(ctx, id)
	if err != nil {
		return err
	}

	if s.hub != nil {
		topic := fmt.Sprintf("worker-%d", id)
		s.hub.PublishAll([]string{topic, "worker-*"}, sse.SSEMessage{
			Event: topic,
			Data:  map[string]int{"id": id},
		})
	}
	return nil
}

// publish publish worker update
func (s *WorkerService) publish(worker domain.Worker) {
	if s.hub == nil {
		return
	}
	topic := fmt.Sprintf("worker-%d", worker.ID)
	s.hub.PublishAll([]string{topic, "worker-*"}, sse.SSEMessage{Event: topic, Data: worker})
}
