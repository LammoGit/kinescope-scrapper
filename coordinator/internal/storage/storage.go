package storage

import (
	"context"
	"coordinator/internal/repos"
)

type Storage interface {
	repos.IJobRepository
	repos.IWorkerRepository
	repos.IVideoRepository
	repos.IEventRepository
	repos.IUserRepository
	repos.ITelemetryRepository
	Shutdown(ctx context.Context) error
}
