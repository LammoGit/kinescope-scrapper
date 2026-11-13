package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
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
func NewUserService(storage st.Storage) (*UserService, error) {
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

	return s.storage.CreateUser(ctx, username, email, domain.PasswordCredentials{
		Hash: hash,
		Salt: salt,
	})
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
	return s.storage.ReplaceUser(ctx, user)
}

// DeleteByID deletes a user with the given id
func (s *UserService) DeleteByID(ctx context.Context, id int) error {
	return s.storage.DeleteUserByID(ctx, id)
}
