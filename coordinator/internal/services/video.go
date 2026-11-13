package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"fmt"
	"strings"
)

// Errors
var (
	ErrVideoStorageUnavailable = utility.NewInternalError("video storage is not configured")
)

// VideoService exposes downloaded videos
type VideoService struct {
	storage st.Storage
	s3URL   string
}

// NewVideoService creates a new VideoService
func NewVideoService(storage st.Storage, s3URL string) (*VideoService, error) {
	if storage == nil {
		return nil, ErrVideoStorageUnavailable
	}
	return &VideoService{storage: storage, s3URL: s3URL}, nil
}

// GetVideoByID returns a video with the given id
func (s *VideoService) GetVideoByID(ctx context.Context, id int) (domain.Video, string, error) {
	video, err := s.storage.GetVideoByID(ctx, id)
	if err != nil {
		return video, "", err
	}
	streamURL := fmt.Sprintf(
		"%s/%s/%s",
		s.s3URL,
		video.StorageMetadata.BucketName,
		strings.TrimLeft(video.StorageMetadata.ObjectKey, "/"),
	)

	return video, streamURL, err
}

// GetVideosByJobID returns every video belonging to a job
func (s *VideoService) GetVideosByJobID(ctx context.Context, jobID int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.storage.GetVideosByJobID(ctx, jobID)
}

// AddVideo stores a video
func (s *VideoService) AddVideo(ctx context.Context, video domain.Video) error {
	return s.storage.AddVideo(ctx, video)
}

// Search returns videos matching the query
func (s *VideoService) Search(ctx context.Context, query string, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideosByQuery(ctx, query, limit, offset)
}

// GetRecommended returns a page of recommended videos
func (s *VideoService) GetRecommended(ctx context.Context, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideos(ctx, limit, offset)
}

// GetRecommendedForVideo returns videos related to the given video
func (s *VideoService) GetRecommendedForVideo(ctx context.Context, videoID, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideos(ctx, limit, offset)
}
