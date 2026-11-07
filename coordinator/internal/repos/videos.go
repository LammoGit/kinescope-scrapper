package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ IVideoRepository = &VideoRepository{}

type IVideoRepository interface {
	GetVideoByID(id int) (domain.Video, bool)
	GetVideosByJobID(jobId int) []domain.Video
	AddVideo(video domain.Video)
}

type VideoRepository struct {
	videos []domain.Video
	m      sync.RWMutex
}

func (vr *VideoRepository) GetVideoByID(id int) (domain.Video, bool)  {}
func (vr *VideoRepository) GetVideosByJobID(jobId int) []domain.Video {}
func (vr *VideoRepository) AddVideo(video domain.Video)               {}

