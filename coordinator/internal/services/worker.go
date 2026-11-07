package services

import (
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"crypto/ed25519"
)

var (
	ErrWorkerNotFoundByID = utility.NewNotFoundError("Worker with the given id doesn't exist")
)

type WorkerService struct{}

func NewWorkerService(hub *sse.Hub, storage st.Storage) (*WorkerService, error) {
	return &WorkerService{}, nil
}

func (s *WorkerService) GetWorkerSignatureKeyByID(id int) (ed25519.PublicKey, error) {
	return nil, nil
}
