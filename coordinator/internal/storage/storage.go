package storage

import (
	"coordinator/internal/repos"
)

type Storage interface {
	repos.IJobRepository
	repos.IRangeRepository
	repos.IWorkerRepository
	repos.IVideoRepository
	repos.IEventRepository
	repos.IUserRepository
	repos.ITelemetryRepository
}
