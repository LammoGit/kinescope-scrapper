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
	ErrEmailIsTaken        = utility.NewConflictError("user with this email already exists")
	ErrUserNotFoundByEmail = utility.NewNotFoundError("user with the given email doesn't exist")
	ErrUserNotFoundByID    = utility.NewNotFoundError("user with the given id doesn't exist")
)

// IUserRepository interface specifying methods that *UserRepository must implement
type IUserRepository interface {
	GetUserByID(ctx context.Context, id int) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	CreateUser(ctx context.Context, username string, email string, creds domain.PasswordCredentials) (domain.User, error)
	ReplaceUser(ctx context.Context, user domain.User) error
	DeleteUserByID(ctx context.Context, id int) error
}

// UserRepository memory storage for User objects
type UserRepository struct {
	users  map[int]domain.User
	nextID int // ID that was given to the newest user
	mu     sync.RWMutex
}

// NewUserRepository creates a new UserRepository object
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users:  make(map[int]domain.User),
		nextID: 1,
	}
}

// GetUserByID returns a user with the given id, or error if wasn't found
func (ur *UserRepository) GetUserByID(ctx context.Context, id int) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	ur.mu.RLock()
	defer ur.mu.RUnlock()

	u, ok := ur.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFoundByID
	}
	return u, nil
}

// GetUserByEmail returns a user with the given email, or error if wasn't found
func (ur *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

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
func (ur *UserRepository) CreateUser(ctx context.Context, username string, email string, creds domain.PasswordCredentials) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

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

func (ur *UserRepository) ReplaceUser(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, ok := ur.users[user.ID]; !ok {
		return ErrUserNotFoundByID
	}

	for _, u := range ur.users {
		if u.Email == user.Email && u.ID != user.ID {
			return ErrEmailIsTaken
		}
	}

	ur.users[user.ID] = user
	return nil
}

// DeleteUserByID deletes user with the specified id
func (ur *UserRepository) DeleteUserByID(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ur.mu.Lock()
	defer ur.mu.Unlock()

	if _, ok := ur.users[id]; ok {
		delete(ur.users, id)
		return nil
	}
	return ErrUserNotFoundByID
}
