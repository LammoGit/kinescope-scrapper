package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"log/slog"
	"time"
)

// Errors
var (
	ErrUserStorageUnavailable = utility.NewInternalError("user storage is not configured")
	ErrBadCredentials         = utility.NewInternalError("invalid credentials were generated during registration")
)

// UserService exposes user account operations on top of the storage layer
type UserService struct {
	storage st.Storage
}

// NewUserService creates a new UserService
func NewUserService(_ context.Context, _ *sse.Hub, storage st.Storage) (*UserService, error) {
	if storage == nil {
		return nil, ErrUserStorageUnavailable
	}
	return &UserService{storage: storage}, nil
}

// Register registers and returns a new User while also validating username and email
func (s *UserService) Register(ctx context.Context, username string, email string, hash []byte, salt []byte) (domain.User, error) {
	if len(hash) == 0 || len(salt) == 0 {
		return domain.User{}, ErrBadCredentials
	}

	user := domain.User{
		ID:       0,
		Username: username,
		Email:    email,
		Password: domain.PasswordCredentials{
			Hash: hash,
			Salt: salt,
		},
		Role:      domain.UserRoleViewer,
		Avatar:    "",
		CreatedAt: time.Now(),
	}
	err := s.storage.SaveUser(ctx, &user)
	if err == nil {
		slog.Info("User registered", "userID", user.ID, "username", user.Username, "email", user.Email)
	}
	return user, err
}

// GetUsers returns every user
func (s *UserService) GetUsers(ctx context.Context) ([]domain.User, error) {
	return s.storage.GetUsers(ctx)
}

// GetUserByEmail returns a user with the given email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.storage.GetUserByEmail(ctx, email)
}

// GetUserByID returns a user with the given id
func (s *UserService) GetUserByID(ctx context.Context, id int) (domain.User, error) {
	return s.storage.GetUserByID(ctx, id)
}

func (s *UserService) ReplaceUser(ctx context.Context, user domain.User) error {
	return s.storage.SaveUser(ctx, &user)
}

// DeleteByID deletes a user with the given id
func (s *UserService) DeleteByID(ctx context.Context, id int) error {
	err := s.storage.DeleteUserByID(ctx, id)
	if err == nil {
		slog.Info("User deleted", "userID", id)
	}
	return err
}
