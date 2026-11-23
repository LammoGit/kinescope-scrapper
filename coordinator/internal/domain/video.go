package domain

import (
	"context"
	"time"
)

// IVideoRepository stores downloaded videos
type IVideoRepository interface {
	AddVideo(ctx context.Context, video *Video) error
	GetVideos(ctx context.Context, limit, offset int) ([]Video, error)
	GetVideoByID(ctx context.Context, id int) (Video, error)
	GetVideosByJobID(ctx context.Context, jobID int) ([]Video, error)
	GetVideosByQuery(ctx context.Context, query string, limit, offset int) ([]Video, error)
}

// VideoMetadata contains the metadata of a downloaded video
type VideoMetadata struct {
	Title        string    `json:"title" form:"title" binding:"required"`
	SourceURL    string    `json:"source_url" form:"source_url" binding:"required,url"`
	DownloadURL  string    `json:"download_url" form:"download_url" binding:"required,url"`
	FoundAt      time.Time `json:"found_at" form:"found_at" binding:"required"`
	DownloadedAt time.Time `json:"downloaded_at" form:"downloaded_at" binding:"required"`
	FileSize     int       `json:"file_size" form:"file_size" binding:"required,gt=0"`
	MimeType     string    `json:"mime_type" form:"mime_type" binding:"required"`
	Status       string    `json:"status" form:"status" binding:"required"`
}

// VideoStorageMetadata holds information used for accessing video in storage
type VideoStorageMetadata struct {
	BucketName string `json:"bucket_name" form:"bucket_name" binding:"required"`
	ObjectKey  string `json:"object_key" form:"object_key" binding:"required"`
	Checksum   string `json:"checksum" form:"checksum" binding:"required,sha256"`
	ServerName string `json:"server_name" form:"server_name" binding:"required"`
	LocalPath  string `json:"local_path" form:"local_path" binding:"required,path"`
}

// Video is a downloaded video belonging to a job
type Video struct {
	ID              int                  `json:"id" form:"id" binding:"required,min=1"`
	JobID           int                  `json:"job_id" form:"job_id" binding:"required,min=1"`
	Metadata        VideoMetadata        `json:"metadata" form:"metadata" binding:"required"`
	StorageMetadata VideoStorageMetadata `json:"storage_metadata" form:"storage_metadata" binding:"required"`
}

// Clone creates a deep-copy of the calling VideoMetadata
func (vm VideoMetadata) Clone() VideoMetadata {
	return vm
}

// Clone creates a deep-copy of the calling VideoStorageMetadata
func (vsm VideoStorageMetadata) Clone() VideoStorageMetadata {
	return vsm
}

// Clone creates a deep-copy of the calling Video
func (v Video) Clone() Video {
	clone := v
	clone.Metadata = v.Metadata.Clone()
	clone.StorageMetadata = v.StorageMetadata.Clone()
	return clone
}
