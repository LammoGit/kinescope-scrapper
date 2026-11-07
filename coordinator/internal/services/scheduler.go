package services

import (
	st "coordinator/internal/storage"
)

type SchedulerService struct {
	storage st.Storage
}

func NewSchedulerService(storage st.Storage) (*SchedulerService, error) {
	return &SchedulerService{storage: storage}, nil
}
