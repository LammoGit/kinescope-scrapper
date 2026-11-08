package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"sync"
	"time"
)

// Won't compile if *UserRepository doesn't implement IUserRepository
var _ IUserRepository = &UserRepository{}

// Errors
var (
	ErrEmailIsTaken        = utility.NewConflictError("User with this email already exists")
	ErrUserNotFoundByEmail = utility.NewNotFoundError("User with the given email doesn't exist")
	ErrUserNotFoundByID    = utility.NewNotFoundError("User with the given id doesn't exist")
)

// IUserRepository interface specifying methods that *UserRepository must implement
type IUserRepository interface {
	GetUserByID(ctx context.Context, id int) (domain.User, error)
	GetUserByEmail(ctx context.Context, email domain.Email) (domain.User, error)
	CreateUser(ctx context.Context, username domain.Username, email domain.Email, creds domain.PasswordCredentials) (domain.User, error)
	UpdateUser(ctx context.Context, id int, update domain.UserUpdate) (domain.User, error)
	DeleteUserByID(ctx context.Context, id int) error
}

// UserRepository memory storage for User objects
type UserRepository struct {
	users  map[int]domain.User
	mu     sync.RWMutex
	nextID int // ID that was given to the newest user
}

// NewUserRepository creates a new UserRepository object
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[int]domain.User),
	}
}

// GetUserByID returns a user with the given id, or error if wasn't found
func (ur *UserRepository) GetUserByID(ctx context.Context, id int) (domain.User, error) {
	ur.mu.RLock()
	defer ur.mu.RUnlock()

	u, ok := ur.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFoundByID
	}
	return u, nil
}

// GetUserByEmail returns a user with the given email, or error if wasn't found
func (ur *UserRepository) GetUserByEmail(ctx context.Context, email domain.Email) (domain.User, error) {
	ur.mu.RLock()
	defer ur.mu.RUnlock()

	for _, u := range ur.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, ErrUserNotFoundByEmail
}

// CreateUser creates a new User object in the repository
func (ur *UserRepository) CreateUser(ctx context.Context, username domain.Username, email domain.Email, creds domain.PasswordCredentials) (domain.User, error) {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	// Check that email is unique
	for _, u := range ur.users {
		if u.Email == email {
			return domain.User{}, ErrEmailIsTaken
		}
	}

	ur.nextID++
	user := domain.User{
		ID:        ur.nextID,
		Username:  username,
		Email:     email,
		Password:  creds,
		Role:      domain.UserRoleViewer,
		CreatedAt: time.Now(),
	}

	ur.users[user.ID] = user
	return user, nil
}

// UpdateUser updates user's fields based on the given values
func (ur *UserRepository) UpdateUser(ctx context.Context, id int, update domain.UserUpdate) (domain.User, error) {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	u, ok := ur.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFoundByID
	}

	if update.Username != nil {
		u.Username = *update.Username
	}

	if update.Email != nil {
		// Check that email is unique
		for _, u := range ur.users {
			if u.Email == *update.Email && u.ID != id {
				return domain.User{}, ErrEmailIsTaken
			}
		}
		u.Email = *update.Email
	}
	if update.Password != nil {
		u.Password = *update.Password
	}
	if update.Avatar != nil {
		u.Avatar = *update.Avatar
	}
	if update.Role != nil {
		u.Role = *update.Role
	}

	ur.users[id] = u
	return u, nil
}

// DeleteUserByID deletes user with the specified id
func (ur *UserRepository) DeleteUserByID(ctx context.Context, id int) error {
	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, ok := ur.users[id]; ok {
		delete(ur.users, id)
		return nil
	}
	return ErrUserNotFoundByID
}
