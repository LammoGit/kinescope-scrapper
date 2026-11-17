//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/repos"
)

var _ Storage = &Memory{}

// Memory is the in-memory implementation of Storage
type Memory struct {
	*repos.JobRepository
	*repos.WorkerRepository
	*repos.VideoRepository
	*repos.EventRepository
	*repos.UserRepository
	*repos.TelemetryRepository
}

// NewStorage creates a fully initialized in-memory storage
func NewStorage() (Storage, error) {
	return &Memory{
		JobRepository:       repos.NewJobRepository(),
		WorkerRepository:    repos.NewWorkerRepository(),
		VideoRepository:     repos.NewVideoRepository(),
		EventRepository:     repos.NewEventRepository(),
		UserRepository:      repos.NewUserRepository(),
		TelemetryRepository: repos.NewTelemetryRepository(),
	}, nil
}

// Shutdown graceful shutdown
func (s *Memory) Shutdown(ctx context.Context) error {
	return nil
}
