package domain

import "time"

type Video struct {
	ID       int           `json:"id" validate:"required"`
	JobID    int           `json:"job_id" validate:"required"`
	Metadata VideoMetadata `json:"metadata" validate:"required"`
	Score    float64       `json:"score"`
	Scores   []int         `json:"scores"`
	Comments []Comment     ``
}

type VideoMetadata struct {
	Title        string    `json:"title" validate:"required"`
	SourceURL    string    `json:"source_url" validate:"required"`
	DownloadURL  string    `json:"download_url" validate:"required"`
	FoundAt      time.Time `json:"found_at" validate:"required"`
	DownloadedAt time.Time `json:"downloaded_at" validate:"required"`
	FileSize     int       `json:"file_size" validate:"required"`
	MimeType     string    `json:"mime_type" validate:"required"`
	Checksum     []byte    `json:"checksum" validate:"required"`
	ServerName   string    `json:"server_name" validate:"required"`
	LocalPath    string    `json:"local_path" validate:"required"`
	Status       string    `json:"status" validate:"required"`
}

type Comment struct {
	UserID    int       `json:"user_id" validate:"required"`
	Content   string    `json:"content" validate:"required"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}
