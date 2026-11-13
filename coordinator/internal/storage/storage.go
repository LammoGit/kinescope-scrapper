package storage

import (
	"coordinator/internal/repos"
)

type Storage interface {
	repos.IJobRepository
	repos.IWorkerRepository
	repos.IVideoRepository
	repos.IEventRepository
	repos.IUserRepository
	repos.ITelemetryRepository
}
