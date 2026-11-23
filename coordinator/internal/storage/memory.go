//go:build !postgres

package storage

import (
	"context"
)

var _ Storage = &Memory{}

// Memory is the in-memory implementation of Storage
type Memory struct {
	*JobRepository
	*WorkerRepository
	*VideoRepository
	*EventRepository
	*UserRepository
	*TelemetryRepository
}

// NewStorage creates a fully initialized in-memory storage
func NewStorage() (Storage, error) {
	return &Memory{
		JobRepository:       NewJobRepository(),
		WorkerRepository:    NewWorkerRepository(),
		VideoRepository:     NewVideoRepository(),
		EventRepository:     NewEventRepository(),
		UserRepository:      NewUserRepository(),
		TelemetryRepository: NewTelemetryRepository(),
	}, nil
}

// Shutdown graceful shutdown
func (s *Memory) Shutdown(ctx context.Context) error {
	return nil
}
