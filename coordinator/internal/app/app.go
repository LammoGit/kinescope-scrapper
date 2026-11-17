package app

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/services"
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Errors
var (
	ErrFailedToGetHomeDir    = errors.New("failed to get home directory")
	ErrFailedToReadPublicKey = errors.New("failed to read public key")
	ErrInvalidPublicKey      = errors.New("read public key is invalid")
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

	// Initialize provisioning service
	{
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, ErrFailedToGetHomeDir
		}

		pubKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519.pub")
		pubBytes, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return nil, ErrFailedToReadPublicKey
		}

		parsedPub, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
		if err != nil {
			return nil, ErrInvalidPublicKey
		}

		pubKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(parsedPub)))

		provisioningService, err := services.NewProvisioningService(ctx, hub, storage, pubKey)
		if err != nil {
			return app, err
		}
		app.Provisioning = provisioningService
		slog.Info("Provisioning service initialized")
	}

	// Initialize cryptography service
	{
		secret := os.Getenv("JWT_SECRET")
		issuer := os.Getenv("JWT_ISSUER")
		argonMemory := os.Getenv("ARGON2_MEMORY")
		argonTime := os.Getenv("ARGON2_ITERATIONS")
		argonThreads := os.Getenv("ARGON2_THREADS")
		keyLen := os.Getenv("PASSWORD_HASH_LEN")
		saltSize := os.Getenv("SALT_SIZE")

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
		cryptoService, err := services.NewCryptoService(ctx, hub, storage, []byte(secret), issuer, opts...)
		if err != nil {
			return app, err
		}
		app.Crypto = cryptoService
		slog.Info("Crypto service initialized")
	}

	// Initialize user service
	{
		userService, err := services.NewUserService(ctx, hub, storage)
		if err != nil {
			return app, err
		}
		app.User = userService
	}

	// Initialize worker service
	{
		workerService, err := services.NewWorkerService(ctx, hub, storage)
		if err != nil {
			return app, err
		}
		app.Worker = workerService
	}

	// Initialize job service
	{
		jobService, err := services.NewJobService(ctx, hub, storage)
		if err != nil {
			return app, err
		}
		app.Job = jobService
	}

	// Initialize events service
	{
		eventService, err := services.NewEventService(ctx, hub, storage)
		if err != nil {
			return app, err
		}
		app.Event = eventService
	}

	// Initialize video service
	{
		s3URL := os.Getenv("S3_SERVER_URL")
		s3AccessKey := os.Getenv("S3_ACCESS_KEY")
		s3SecretKey := os.Getenv("S3_SECRET_KEY")
		s3Region := os.Getenv("S3_REGION")
		s3Bucket := os.Getenv("S3_BUCKET")

		videoService, err := services.NewVideoService(ctx, hub, storage, s3URL, s3Region, s3Bucket, s3AccessKey, s3SecretKey)
		if err != nil {
			return app, err
		}
		app.Video = videoService
	}
	// Initialize telemetry service
	{
		telemetryService, err := services.NewTelemetryService(ctx, hub, storage)
		if err != nil {
			return app, err
		}
		app.Telemetry = telemetryService
	}

	// Initialize administrator
	{
		adminUsername := os.Getenv("ADMIN_USERNAME")
		adminEmail := os.Getenv("ADMIN_EMAIL")
		adminPassword := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
		hash, salt, err := app.Crypto.HashPassword(adminPassword)
		if err != nil {
			return app, err
		}

		// Try to create an admin account
		if _, err := storage.CreateUser(ctx, adminUsername, adminEmail, domain.PasswordCredentials{
			Hash: hash,
			Salt: salt,
		}, domain.UserRoleAdministrator); err == nil {
			slog.Info("Administrator account created", "username", adminUsername, "email", adminEmail)
		}
	}

	return app, nil
}

func (a *App) Shutdown(ctx context.Context) (err error) {
	return err
}
