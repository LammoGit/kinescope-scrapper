package domain

import "time"

// Video is a downloaded video belonging to a job
type Video struct {
	ID       int           `json:"id" form:"id" binding:"min=0"`
	JobID    int           `json:"job_id" form:"job_id" binding:"min=0,required"`
	Metadata VideoMetadata `json:"metadata" form:"metadata" binding:"required"`
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
	Checksum     string    `json:"checksum" form:"checksum" binding:"required,sha256"`
	ServerName   string    `json:"server_name" form:"server_name" binding:"required"`
	LocalPath    string    `json:"local_path" form:"local_path" binding:"required,filepath"`
	Status       string    `json:"status" form:"status" binding:"required"`
}
