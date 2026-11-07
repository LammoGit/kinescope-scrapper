package services

import (
	st "coordinator/internal/storage"
)

type VideoService struct{}

func NewVideoService(storage st.Storage) (*VideoService, error) {
	return &VideoService{}, nil
}
