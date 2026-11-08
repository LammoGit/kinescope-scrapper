package repos

import (
	"coordinator/internal/domain"
	"sync"
)

var _ IVideoRepository = &VideoRepository{}

// IVideoRepository stores downloaded videos
type IVideoRepository interface {
	GetVideos() []domain.Video
	GetVideoByID(id int) (domain.Video, bool)
	GetVideosByJobID(jobID int) []domain.Video
	AddVideo(video domain.Video) domain.Video
}

// VideoRepository is an in-memory video store
type VideoRepository struct {
	videos []domain.Video
	m      sync.RWMutex
	nextID int
}

// NewVideoRepository creates a new VideoRepository
func NewVideoRepository() *VideoRepository {
	return &VideoRepository{videos: make([]domain.Video, 0)}
}

// GetVideos returns every stored video in insertion order
func (vr *VideoRepository) GetVideos() []domain.Video {
	vr.m.RLock()
	defer vr.m.RUnlock()

	res := make([]domain.Video, len(vr.videos))
	copy(res, vr.videos)
	return res
}

// GetVideoByID returns a video by its identifier
func (vr *VideoRepository) GetVideoByID(id int) (domain.Video, bool) {
	vr.m.RLock()
	defer vr.m.RUnlock()

	for _, video := range vr.videos {
		if video.ID == id {
			return video, true
		}
	}
	return domain.Video{}, false
}

// GetVideosByJobID returns every video belonging to a job
func (vr *VideoRepository) GetVideosByJobID(jobID int) []domain.Video {
	vr.m.RLock()
	defer vr.m.RUnlock()

	var res []domain.Video
	for _, video := range vr.videos {
		if video.JobID == jobID {
			res = append(res, video)
		}
	}
	return res
}

// AddVideo stores a video and returns it
func (vr *VideoRepository) AddVideo(video domain.Video) domain.Video {
	vr.m.Lock()
	defer vr.m.Unlock()

	if video.ID == 0 {
		vr.nextID++
		video.ID = vr.nextID
	} else if video.ID > vr.nextID {
		vr.nextID = video.ID
	}
	vr.videos = append(vr.videos, video)
	return video
}
