package storage_test

import (
	"bytes"
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	crand "crypto/rand"
	"errors"
	"math/rand/v2"
	"testing"
	"time"
)

func randomUser() domain.User {
	return domain.User{
		ID:       rand.Int(),
		Username: crand.Text(),
		Email:    crand.Text(),
		Password: domain.PasswordCredentials{
			Hash: []byte(crand.Text()),
			Salt: []byte(crand.Text()),
		},
		Role:      crand.Text(),
		Avatar:    crand.Text(),
		CreatedAt: time.UnixMilli(rand.Int64()),
	}
}

func equalUsers(u1, u2 domain.User) bool {
	return u1.ID == u2.ID &&
		u1.Username == u2.Username &&
		u1.Email == u2.Email &&
		u1.Avatar == u2.Avatar &&
		u1.CreatedAt.Equal(u2.CreatedAt) &&
		u1.Role == u2.Role &&
		bytes.Equal(u1.Password.Hash, u2.Password.Hash) &&
		bytes.Equal(u1.Password.Salt, u2.Password.Salt)
}

func TestUsers(t *testing.T) {
	t.Run("Save and get a user", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		user := randomUser()
		user.ID = 0

		err = s.SaveUser(ctx, &user)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		{
			stored, err := s.GetUsers(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || !equalUsers(stored[0], user) {
				t.Fatalf("Received user isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetUserByID(ctx, user.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !equalUsers(stored, user) {
				t.Fatalf("Received user isn't equal to the saved one, got %v, want %v", stored, user)
			}
		}

		{
			stored, err := s.GetUserByEmail(ctx, user.Email)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !equalUsers(stored, user) {
				t.Fatalf("Received user isn't equal to the saved one, got %v, want %v", stored, user)
			}
		}
	})

	t.Run("Create with taken email", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		user := randomUser()
		user.ID = 0

		err = s.SaveUser(ctx, &user)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		user.ID = 0
		err = s.SaveUser(ctx, &user)
		if !errors.Is(err, storage.ErrEmailIsTaken) {
			t.Fatalf("Unexpected error, got: %v, want: %v", err, storage.ErrEmailIsTaken)
		}
	})

	t.Run("Create with taken email", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		user := randomUser()
		user.ID = 0

		err = s.SaveUser(ctx, &user)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		user2 := randomUser()
		user2.ID = 0

		err = s.SaveUser(ctx, &user2)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		user.Email = user2.Email

		err = s.SaveUser(ctx, &user)
		if !errors.Is(err, storage.ErrEmailIsTaken) {
			t.Fatalf("Unexpected error, got: %v, want: %v", err, storage.ErrEmailIsTaken)
		}
	})

	t.Run("Save, update and get a user", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		user := randomUser()
		user.ID = 0

		err = s.SaveUser(ctx, &user)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		user2 := user.Clone()
		user2.Email = user.Email + "123"

		err = s.SaveUser(ctx, &user2)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		stored, err := s.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if equalUsers(stored, user) {
			t.Fatalf("Received user is equal to the previous version, got %v, want %v", stored, user2)
		}

		if !equalUsers(stored, user2) {
			t.Fatalf("Received user isn't equal to either saved user, got %v, want %v", stored, user2)
		}
	})

	t.Run("Save, delete and get a user", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		user := randomUser()
		user.ID = 0

		err = s.SaveUser(ctx, &user)
		if err != nil {
			t.Fatalf("Failed to save a user: %v", err)
		}

		err = s.DeleteUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to delete a user: %v", err)
		}

		{
			stored, err := s.GetUsers(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received user isn't equal to the saved one, got %v", stored)
			}
		}

		{
			_, err := s.GetUserByID(ctx, user.ID)
			if !errors.Is(err, storage.ErrUserNotFoundByID) {
				t.Fatalf("Unexpected error, got: %v, expected: %v", err, storage.ErrUserNotFoundByID)
			}
		}

		{
			_, err := s.GetUserByEmail(ctx, user.Email)
			if !errors.Is(err, storage.ErrUserNotFoundByEmail) {
				t.Fatalf("Unexpected error, got: %v, expected: %v", err, storage.ErrUserNotFoundByEmail)
			}
		}
	})
}
