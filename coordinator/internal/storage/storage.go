package storage

import (
	"context"
	"coordinator/internal/domain"
)

type Storage interface {
	domain.IJobRepository
	domain.IWorkerRepository
	domain.IVideoRepository
	domain.IEventRepository
	domain.IUserRepository
	domain.ITelemetryRepository
	Shutdown(ctx context.Context) error
}
