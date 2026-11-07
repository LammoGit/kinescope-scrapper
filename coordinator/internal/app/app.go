package app

import (
	"coordinator/internal/services"
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
	"os"
	"strconv"
)

type App struct {
	Hub          *sse.Hub
	Provisioning *services.ProvisioningService
	Crypto       *services.CryptoService
	Scheduler    *services.SchedulerService
	User         *services.UserService
	Worker       *services.WorkerService
	Job          *services.JobService
	Event        *services.EventService
	Range        *services.RangeService
	Video        *services.VideoService
	Telemetry    *services.TelemetryService
}

func New(hub *sse.Hub, storage st.Storage) *App {
	var cryptoService *services.CryptoService

	secret := os.Getenv("JWT_SECRET")
	issuer := os.Getenv("JWT_ISSUER")
	argonMemory := os.Getenv("ARGON2_MEMORY")
	argonTime := os.Getenv("ARGON2_ITERATIONS")
	argonThreads := os.Getenv("ARGON2_THREADS")
	keyLen := os.Getenv("PASSWORD_HASH_LEN")
	saltSize := os.Getenv("SALT_SIZE")

	provisioningService, _ := services.NewProvisioningService()

	{
		var opts []services.CryptoOption
		if val, err := strconv.Atoi(argonMemory); err == nil {
			opts = append(opts, services.WithArgonMemory(uint32(val)))
		}
		if val, err := strconv.Atoi(argonTime); err == nil {
			opts = append(opts, services.WithArgonIterations(uint32(val)))
		}
		if val, err := strconv.Atoi(argonThreads); err == nil {
			opts = append(opts, services.WithArgonThreads(uint8(val)))
		}
		if val, err := strconv.Atoi(keyLen); err == nil {
			opts = append(opts, services.WithKeyLen(uint32(val)))
		}
		if val, err := strconv.Atoi(saltSize); err == nil {
			opts = append(opts, services.WithSaltSize(val))
		}
		cryptoService, _ = services.NewCryptoService([]byte(secret), issuer, opts...)
	}

	schedulerService, _ := services.NewSchedulerService(storage)

	userService, _ := services.NewUserService(storage)

	workerService, _ := services.NewWorkerService(hub, storage)

	jobService, _ := services.NewJobService(hub, storage)

	eventService, _ := services.NewEventService(hub, storage)

	rangeService, _ := services.NewRangeService(storage)

	videoService, _ := services.NewVideoService(storage)

	telemetryService, _ := services.NewTelemetryService(hub, storage)

	return &App{
		Hub:          hub,
		Provisioning: provisioningService,
		Crypto:       cryptoService,
		Scheduler:    schedulerService,
		User:         userService,
		Worker:       workerService,
		Job:          jobService,
		Event:        eventService,
		Range:        rangeService,
		Video:        videoService,
		Telemetry:    telemetryService,
	}
}
