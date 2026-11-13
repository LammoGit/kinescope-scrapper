package app

import (
	"context"
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
	User         *services.UserService
	Worker       *services.WorkerService
	Job          *services.JobService
	Event        *services.EventService
	Video        *services.VideoService
	Telemetry    *services.TelemetryService
	ctx          context.Context
}

func New(ctx context.Context, hub *sse.Hub, storage st.Storage) (*App, error) {
	app := &App{
		Hub: hub,
		ctx: ctx,
	}

	secret := os.Getenv("JWT_SECRET")
	issuer := os.Getenv("JWT_ISSUER")
	argonMemory := os.Getenv("ARGON2_MEMORY")
	argonTime := os.Getenv("ARGON2_ITERATIONS")
	argonThreads := os.Getenv("ARGON2_THREADS")
	keyLen := os.Getenv("PASSWORD_HASH_LEN")
	saltSize := os.Getenv("SALT_SIZE")
	s3URL := os.Getenv("S3_SERVER_URL")

	provisioningService, err := services.NewProvisioningService()
	if err != nil {
		return app, err
	}
	app.Provisioning = provisioningService

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
		cryptoService, err := services.NewCryptoService([]byte(secret), issuer, opts...)
		if err != nil {
			return app, err
		}
		app.Crypto = cryptoService

	}

	userService, err := services.NewUserService(storage)
	if err != nil {
		return app, err
	}
	app.User = userService

	workerService, err := services.NewWorkerService(hub, storage)
	if err != nil {
		return app, err
	}
	app.Worker = workerService

	jobService, err := services.NewJobService(ctx, hub, storage)
	if err != nil {
		return app, err
	}
	app.Job = jobService

	eventService, err := services.NewEventService(hub, storage)
	if err != nil {
		return app, err
	}
	app.Event = eventService

	videoService, err := services.NewVideoService(storage, s3URL)
	if err != nil {
		return app, err
	}
	app.Video = videoService

	telemetryService, err := services.NewTelemetryService(hub, storage)
	if err != nil {
		return app, err
	}
	app.Telemetry = telemetryService

	return app, nil
}
