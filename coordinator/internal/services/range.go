package services

import (
	st "coordinator/internal/storage"
)

type RangeService struct{}

func NewRangeService(storage st.Storage) (*RangeService, error) {
	return &RangeService{}, nil
}
