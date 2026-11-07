package services

import (
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
)

type JobService struct{}

func NewJobService(hub *sse.Hub, storage st.Storage) (*JobService, error) {
	return &JobService{}, nil
}
