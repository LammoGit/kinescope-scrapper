package repos

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"strings"
	"sync"
)

// Won't compile if *VideoRepository doesn't implement IVideoRepository
var _ IVideoRepository = &VideoRepository{}

// Errors
var (
	ErrVideoNotFoundByID = utility.NewNotFoundError("video with given id doesn't exist")
)

// IVideoRepository stores downloaded videos
type IVideoRepository interface {
	AddVideo(ctx context.Context, video domain.Video) error
	GetVideos(ctx context.Context, limit, offset int) ([]domain.Video, error)
	GetVideoByID(ctx context.Context, id int) (domain.Video, error)
	GetVideosByJobID(ctx context.Context, jobID int) ([]domain.Video, error)
	GetVideosByQuery(ctx context.Context, query string, limit, offset int) ([]domain.Video, error)
}

// VideoRepository is an in-memory video store
type VideoRepository struct {
	videos map[int]domain.Video
	nextID int
	mu     sync.RWMutex
}

// NewVideoRepository creates a new VideoRepository
func NewVideoRepository() *VideoRepository {
	return &VideoRepository{
		videos: make(map[int]domain.Video),
		nextID: 1,
	}
}

// AddVideo stores a video and returns it
func (vr *VideoRepository) AddVideo(ctx context.Context, video domain.Video) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	vr.mu.Lock()
	defer vr.mu.Unlock()

	vr.nextID++
	video.ID = vr.nextID
	vr.videos[vr.nextID] = video
	return nil
}

// GetVideos returns stored videos
func (vr *VideoRepository) GetVideos(ctx context.Context, limit, offset int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	vr.mu.RLock()
	defer vr.mu.RUnlock()

	if limit <= 0 || offset < 0 || offset >= len(vr.videos) {
		return []domain.Video{}, nil
	}

	res := make([]domain.Video, 0, min(limit, len(vr.videos)-offset))

	for i := 0; i < vr.nextID; i++ {
		video, ok := vr.videos[i]
		if !ok {
			continue
		}

		if offset > 0 {
			offset--
			continue
		}

		res = append(res, video)
		limit -= 1
		if limit <= 0 {
			break
		}
	}
	return res, nil
}

// GetVideoByID returns a video by its identifier
func (vr *VideoRepository) GetVideoByID(ctx context.Context, id int) (domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return domain.Video{}, err
	}

	vr.mu.RLock()
	defer vr.mu.RUnlock()

	video, ok := vr.videos[id]
	if !ok {
		return video, ErrVideoNotFoundByID
	}
	return video, nil
}

// GetVideosByJobID returns every video belonging to a job
func (vr *VideoRepository) GetVideosByJobID(ctx context.Context, jobID int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	vr.mu.RLock()
	defer vr.mu.RUnlock()

	var res []domain.Video
	for _, video := range vr.videos {
		if video.JobID == jobID {
			res = append(res, video)
		}
	}
	return res, nil
}

// GetVideosByQuery returns videos by user query with given limit and offset
func (vr *VideoRepository) GetVideosByQuery(ctx context.Context, query string, limit, offset int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var res []domain.Video
	for _, v := range vr.videos {
		if strings.Contains(v.Metadata.Title, query) {
			if offset > 0 {
				offset -= 1
				continue
			}

			res = append(res, v)

			limit -= 1
			if limit <= 0 {
				break
			}
		}
	}
	return res, nil
}
