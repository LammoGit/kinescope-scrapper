package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ IRangeRepository = &RangeRepository{}

type IRangeRepository interface {
}

type RangeRepository struct {
	ranges map[int]domain.Range
	m      sync.RWMutex
}
