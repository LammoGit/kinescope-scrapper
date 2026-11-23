package storage_test

import (
	"context"
	"coordinator/internal/domain"
	"coordinator/internal/storage"
	crand "crypto/rand"
	"errors"
	"math/rand/v2"
	"testing"
	"time"
)

func randomVideo() domain.Video {
	return domain.Video{
		ID:    rand.Int(),
		JobID: rand.Int(),
		Metadata: domain.VideoMetadata{
			Title:        crand.Text(),
			SourceURL:    crand.Text(),
			DownloadURL:  crand.Text(),
			FoundAt:      time.UnixMilli(rand.Int64()),
			DownloadedAt: time.UnixMilli(rand.Int64()),
			FileSize:     rand.Int(),
			MimeType:     crand.Text(),
			Status:       crand.Text(),
		},
		StorageMetadata: domain.VideoStorageMetadata{
			BucketName: crand.Text(),
			ObjectKey:  crand.Text(),
			Checksum:   crand.Text(),
			ServerName: crand.Text(),
			LocalPath:  crand.Text(),
		},
	}
}

func TestVideos(t *testing.T) {
	t.Run("Save and get a video", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()

		video := randomVideo()
		video.ID = 0

		err = s.AddVideo(ctx, &video)
		if err != nil {
			t.Fatalf("Failed to save a video: %v", err)
		}

		{
			stored, err := s.GetVideos(ctx, 100, 0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != video {
				t.Fatalf("Received video isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetVideos(ctx, 100, 1)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received video isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetVideoByID(ctx, video.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if stored != video {
				t.Fatalf("Received video isn't equal to the saved one, got %v, want %v", stored, video)
			}
		}

		{
			stored, err := s.GetVideosByJobID(ctx, video.JobID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != video {
				t.Fatalf("Received video isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetVideosByQuery(ctx, video.Metadata.Title, 100, 0)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 1 || stored[0] != video {
				t.Fatalf("Received video isn't equal to the saved one, got %v", stored)
			}
		}

		{
			stored, err := s.GetVideosByQuery(ctx, "", 100, 1)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(stored) != 0 {
				t.Fatalf("Received video isn't equal to the saved one, got %v", stored)
			}
		}
	})

	t.Run("GetVideoByID request non-existant video", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		ctx := context.Background()
		_, err = s.GetVideoByID(ctx, 0)
		if err != storage.ErrVideoNotFoundByID {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Test passing context with error", func(t *testing.T) {
		s, err := storage.NewStorage()
		if err != nil {
			t.Fatalf("Failed to start a storage: %v", err)
		}

		err = errors.New("Test error")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(err)

		{
			err := s.AddVideo(ctx, nil)
			if err != ctx.Err() {
				t.Fatalf("Received unexpected error, got: %v, expected: %v", err, ctx.Err())
			}
		}

		{
			_, err := s.GetVideos(ctx, 100, 0)
			if err != ctx.Err() {
				t.Fatalf("Received unexpected error, got: %v, expected: %v", err, ctx.Err())
			}
		}

		{
			_, err := s.GetVideoByID(ctx, 1)
			if err != ctx.Err() {
				t.Fatalf("Received unexpected error, got: %v, expected: %v", err, ctx.Err())
			}
		}

		{
			_, err := s.GetVideosByJobID(ctx, 1)
			if err != ctx.Err() {
				t.Fatalf("Received unexpected error, got: %v, expected: %v", err, ctx.Err())
			}
		}

		{
			_, err := s.GetVideosByQuery(ctx, "", 100, 0)
			if err != ctx.Err() {
				t.Fatalf("Received unexpected error, got: %v, expected: %v", err, ctx.Err())
			}
		}
	})
}
