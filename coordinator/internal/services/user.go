package services

import (
	st "coordinator/internal/storage"
)

type UserService struct{}

func NewUserService(storage st.Storage) (*UserService, error) {
	return &UserService{}, nil
}
