//go:build !postgres && !sqlite

package storage

import "coordinator/internal/repos"

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
func NewStorage() *Memory {
	return &Memory{
		JobRepository:       repos.NewJobRepository(),
		WorkerRepository:    repos.NewWorkerRepository(),
		VideoRepository:     repos.NewVideoRepository(),
		EventRepository:     repos.NewEventRepository(),
		UserRepository:      repos.NewUserRepository(),
		TelemetryRepository: repos.NewTelemetryRepository(),
	}
}
