//go:build !postgres

package storage

import (
	"context"
	"coordinator/internal/domain"
	"sort"
	"sync"
)

// Won't compile if *UserRepository doesn't implement IUserRepository
var _ domain.IUserRepository = &UserRepository{}

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

// SaveUser adds new user if id is 0 and updates otherwise.
func (ur *UserRepository) SaveUser(ctx context.Context, user *domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ur.mu.Lock()
	defer ur.mu.Unlock()

	// Insert
	if user.ID == 0 {
		for _, u := range ur.users {
			if u.Email == user.Email {
				return ErrEmailIsTaken
			}
		}

		user.ID = ur.nextID
		ur.nextID++

		ur.users[user.ID] = user.Clone()
		return nil
	}

	// Update
	if _, ok := ur.users[user.ID]; !ok {
		return ErrUserNotFoundByID
	}

	for _, u := range ur.users {
		if u.Email == user.Email && u.ID != user.ID {
			return ErrEmailIsTaken
		}
	}

	ur.users[user.ID] = user.Clone()
	return nil
}

// GetUsers returns every user, ordered by id
func (ur *UserRepository) GetUsers(ctx context.Context) ([]domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ur.mu.RLock()
	defer ur.mu.RUnlock()

	users := make([]domain.User, 0, len(ur.users))
	for _, u := range ur.users {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	return users, nil
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
