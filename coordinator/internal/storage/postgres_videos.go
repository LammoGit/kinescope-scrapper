//go:build postgres && !sqlite

package storage

import (
	"context"

	"coordinator/internal/domain"
)

const videoColumns = `
	id, job_id, title, source_url, download_url, found_at, downloaded_at,
	file_size, mime_type, status, bucket_name, object_key, checksum,
	server_name, local_path`

func scanVideo(s rowScanner) (domain.Video, error) {
	var v domain.Video
	err := s.Scan(
		&v.ID, &v.JobID, &v.Metadata.Title, &v.Metadata.SourceURL,
		&v.Metadata.DownloadURL, &v.Metadata.FoundAt, &v.Metadata.DownloadedAt,
		&v.Metadata.FileSize, &v.Metadata.MimeType, &v.Metadata.Status,
		&v.StorageMetadata.BucketName, &v.StorageMetadata.ObjectKey,
		&v.StorageMetadata.Checksum, &v.StorageMetadata.ServerName,
		&v.StorageMetadata.LocalPath,
	)
	return v, err
}

func (p *Postgres) AddVideo(ctx context.Context, video domain.Video) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO videos (
			job_id, title, source_url, download_url, found_at, downloaded_at,
			file_size, mime_type, status, bucket_name, object_key, checksum,
			server_name, local_path
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		video.JobID, video.Metadata.Title, video.Metadata.SourceURL,
		video.Metadata.DownloadURL, video.Metadata.FoundAt, video.Metadata.DownloadedAt,
		video.Metadata.FileSize, video.Metadata.MimeType, video.Metadata.Status,
		video.StorageMetadata.BucketName, video.StorageMetadata.ObjectKey,
		video.StorageMetadata.Checksum, video.StorageMetadata.ServerName,
		video.StorageMetadata.LocalPath,
	)
	return err
}

func (p *Postgres) GetVideos(ctx context.Context, limit, offset int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || offset < 0 {
		return []domain.Video{}, nil
	}

	return p.queryVideos(ctx,
		`SELECT `+videoColumns+` FROM videos ORDER BY id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
}

func (p *Postgres) GetVideoByID(ctx context.Context, id int) (domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return domain.Video{}, err
	}

	video, err := scanVideo(p.db.QueryRowContext(ctx, `SELECT `+videoColumns+` FROM videos WHERE id = $1`, id))
	return video, withNotFound(err, errVideoNotFound)
}

func (p *Postgres) GetVideosByJobID(ctx context.Context, jobID int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.queryVideos(ctx,
		`SELECT `+videoColumns+` FROM videos WHERE job_id = $1 ORDER BY id`,
		jobID,
	)
}

func (p *Postgres) GetVideosByQuery(ctx context.Context, query string, limit, offset int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || offset < 0 {
		return []domain.Video{}, nil
	}

	return p.queryVideos(ctx, `
		SELECT `+videoColumns+`
		FROM videos
		WHERE strpos(title, $1) > 0
		ORDER BY id
		LIMIT $2 OFFSET $3`,
		query, limit, offset,
	)
}

func (p *Postgres) queryVideos(ctx context.Context, query string, args ...any) ([]domain.Video, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []domain.Video{}
	for rows.Next() {
		video, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}
	return videos, rows.Err()
}
