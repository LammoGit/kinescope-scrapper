//go:build postgres

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"coordinator/internal/domain"
)

const jobColumns = `
	id, name, range_from, range_to, request_timeout, max_tries,
	global_rate_limit_rps, stop_on_consecutive_missing, status, paused,
	expiration, workers, servers`

const rangeColumns = `
	id, job_id, range_start, range_end, next_id, status, leased_by,
	lease_expires_at, attempt, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func scanJob(s rowScanner) (domain.Job, error) {
	var (
		job        domain.Job
		workersRaw []byte
		serversRaw []byte
	)

	err := s.Scan(
		&job.ID, &job.Config.Name, &job.Config.Range.From, &job.Config.Range.To,
		&job.Config.RequestTimeout, &job.Config.MaxTries, &job.Config.GlobalRateLimitRPS,
		&job.Config.StopOnConsecutiveMissing, &job.Status, &job.Paused, &job.Expiration,
		&workersRaw, &serversRaw,
	)
	if err != nil {
		return domain.Job{}, err
	}

	job.Workers = []int{}
	if len(workersRaw) > 0 {
		if err := json.Unmarshal(workersRaw, &job.Workers); err != nil {
			return domain.Job{}, fmt.Errorf("decode job workers: %w", err)
		}
	}
	job.Config.Servers = []domain.WorkerConfig{}
	if len(serversRaw) > 0 {
		if err := json.Unmarshal(serversRaw, &job.Config.Servers); err != nil {
			return domain.Job{}, fmt.Errorf("decode job servers: %w", err)
		}
	}

	return job, nil
}

func scanRange(s rowScanner) (domain.Range, error) {
	var (
		r     domain.Range
		lease sql.NullTime
	)

	err := s.Scan(
		&r.ID, &r.JobID, &r.RangeStart, &r.RangeEnd, &r.NextID, &r.Status,
		&r.LeasedBy, &lease, &r.Attempt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return domain.Range{}, err
	}
	if lease.Valid {
		r.LeaseExpiresAt = lease.Time
	}

	return r, nil
}

func collectRanges(rows *sql.Rows) ([]domain.Range, error) {
	ranges := []domain.Range{}
	for rows.Next() {
		r, err := scanRange(rows)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, r)
	}
	return ranges, rows.Err()
}

func (p *Postgres) checkRangeConflict(ctx context.Context, jobID int, from, to int64) error {
	var conflict bool
	err := p.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM jobs
			WHERE id <> $1
			  AND ((range_from <= $2 AND $2 <= range_to)
			       OR (range_from <= $3 AND $3 <= range_to))
		)
	`, jobID, from, to).Scan(&conflict)
	if err != nil {
		return err
	}
	if conflict {
		return domain.ErrRangeIsTaken
	}
	return nil
}

// SaveJob adds new job if id is 0 and updates otherwise.
func (p *Postgres) SaveJob(ctx context.Context, job *domain.Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := p.checkRangeConflict(ctx, job.ID, int64(job.Config.Range.From), int64(job.Config.Range.To)); err != nil {
		return err
	}

	workers, err := jsonBytes(job.Workers)
	if err != nil {
		return err
	}
	servers, err := jsonBytes(job.Config.Servers)
	if err != nil {
		return err
	}

	if job.ID == 0 {
		var id int
		err := p.db.QueryRowContext(ctx, `
			INSERT INTO jobs (
				name, range_from, range_to, request_timeout, max_tries,
				global_rate_limit_rps, stop_on_consecutive_missing, status,
				paused, expiration, workers, servers
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id`,
			job.Config.Name, job.Config.Range.From, job.Config.Range.To,
			job.Config.RequestTimeout, job.Config.MaxTries, job.Config.GlobalRateLimitRPS,
			job.Config.StopOnConsecutiveMissing, job.Status, job.Paused,
			nullableTime(job.Expiration), workers, servers,
		).Scan(&id)
		if err != nil {
			return err
		}
		job.ID = id
		if job.Config.Name == "" {
			job.Config.Name = fmt.Sprintf("job-%d", id)
			if _, err := p.db.ExecContext(ctx, `UPDATE jobs SET name = $1 WHERE id = $2`, job.Config.Name, id); err != nil {
				return err
			}
		}
		return nil
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE jobs SET
			name = $2, range_from = $3, range_to = $4, request_timeout = $5,
			max_tries = $6, global_rate_limit_rps = $7,
			stop_on_consecutive_missing = $8, status = $9, paused = $10,
			expiration = $11, workers = $12, servers = $13, updated_at = now()
		WHERE id = $1`,
		job.ID, job.Config.Name, job.Config.Range.From, job.Config.Range.To,
		job.Config.RequestTimeout, job.Config.MaxTries, job.Config.GlobalRateLimitRPS,
		job.Config.StopOnConsecutiveMissing, job.Status, job.Paused,
		nullableTime(job.Expiration), workers, servers,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrJobNotFoundByID
	}
	return nil
}

// GetJobByID returns a job by its identifier.
func (p *Postgres) GetJobByID(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	job, err := scanJob(p.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id))
	if err != nil {
		return domain.Job{}, withNotFound(err, ErrJobNotFoundByID)
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT `+rangeColumns+` FROM ranges WHERE job_id = $1 ORDER BY range_start, id`,
		id,
	)
	if err != nil {
		return domain.Job{}, err
	}
	defer rows.Close()

	ranges, err := collectRanges(rows)
	if err != nil {
		return domain.Job{}, err
	}
	job.Ranges = ranges
	return job, nil
}

// GetJobs returns every stored job ordered by identifier
func (p *Postgres) GetJobs(ctx context.Context) ([]domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	jobRows, err := p.db.QueryContext(ctx, `SELECT `+jobColumns+` FROM jobs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer jobRows.Close()

	jobs := []domain.Job{}
	for jobRows.Next() {
		job, err := scanJob(jobRows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := jobRows.Err(); err != nil {
		return nil, err
	}

	rangeRows, err := p.db.QueryContext(ctx, `SELECT `+rangeColumns+` FROM ranges ORDER BY job_id, range_start, id`)
	if err != nil {
		return nil, err
	}
	defer rangeRows.Close()

	rangesByJob := make(map[int][]domain.Range)
	for rangeRows.Next() {
		r, err := scanRange(rangeRows)
		if err != nil {
			return nil, err
		}
		rangesByJob[r.JobID] = append(rangesByJob[r.JobID], r)
	}
	if err := rangeRows.Err(); err != nil {
		return nil, err
	}

	for i := range jobs {
		jobs[i].Ranges = rangesByJob[jobs[i].ID]
	}
	return jobs, nil
}

// DeleteJobByID removes a job
func (p *Postgres) DeleteJobByID(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := p.db.ExecContext(ctx, `DELETE FROM ranges WHERE job_id = $1`, id); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrJobNotFoundByID
	}
	return nil
}

// SaveRange adds new range if id is 0 and updates otherwise
func (p *Postgres) SaveRange(ctx context.Context, rangeBlock *domain.Range) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var exists bool
	if err := p.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM jobs WHERE id = $1)`, rangeBlock.JobID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrJobNotFoundByID
	}

	if rangeBlock.ID == 0 {
		now := time.Now()
		if rangeBlock.CreatedAt.IsZero() {
			rangeBlock.CreatedAt = now
		}
		if rangeBlock.NextID == 0 {
			rangeBlock.NextID = rangeBlock.RangeStart
		}
		if rangeBlock.Status == "" {
			rangeBlock.Status = domain.JobStatusCreated
		}
		rangeBlock.UpdatedAt = now
		rangeBlock.LeasedBy = 0

		var id int
		err := p.db.QueryRowContext(ctx, `
			INSERT INTO ranges (
				job_id, range_start, range_end, next_id, status, leased_by,
				lease_expires_at, attempt, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			rangeBlock.JobID, rangeBlock.RangeStart, rangeBlock.RangeEnd,
			rangeBlock.NextID, rangeBlock.Status, rangeBlock.LeasedBy,
			nullableTime(rangeBlock.LeaseExpiresAt), rangeBlock.Attempt,
			rangeBlock.CreatedAt, rangeBlock.UpdatedAt,
		).Scan(&id)
		if err != nil {
			return err
		}
		rangeBlock.ID = id
		return nil
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE ranges SET
			job_id = $2, range_start = $3, range_end = $4, next_id = $5,
			status = $6, leased_by = $7, lease_expires_at = $8, attempt = $9,
			created_at = $10, updated_at = $11
		WHERE id = $1`,
		rangeBlock.ID, rangeBlock.JobID, rangeBlock.RangeStart, rangeBlock.RangeEnd,
		rangeBlock.NextID, rangeBlock.Status, rangeBlock.LeasedBy,
		nullableTime(rangeBlock.LeaseExpiresAt), rangeBlock.Attempt,
		rangeBlock.CreatedAt, rangeBlock.UpdatedAt,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrRangeNotFoundByID
	}
	return nil
}

// GetRangeByID returns a copy of a range by its identifier
func (p *Postgres) GetRangeByID(ctx context.Context, id int) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	r, err := scanRange(p.db.QueryRowContext(ctx, `SELECT `+rangeColumns+` FROM ranges WHERE id = $1`, id))
	return r, withNotFound(err, ErrRangeNotFoundByID)
}

// GetRangesByJobID returns all ranges belonging to a job ordered by start
func (p *Postgres) GetRangesByJobID(ctx context.Context, jobID int) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT `+rangeColumns+` FROM ranges WHERE job_id = $1 ORDER BY range_start, id`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectRanges(rows)
}

// GetRangesByWorkerID
func (p *Postgres) GetRangesByWorkerID(ctx context.Context, workerID int) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx,
		`SELECT `+rangeColumns+` FROM ranges WHERE leased_by = $1 ORDER BY job_id, range_start, id`,
		workerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectRanges(rows)
}

// RenewLeases renews all ranges leased by a worker
func (p *Postgres) RenewLeases(ctx context.Context, workerID int, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := p.db.ExecContext(ctx, `
		UPDATE ranges
		SET lease_expires_at = now() + make_interval(secs => $2), updated_at = now()
		WHERE leased_by = $1 AND status = 'running' AND lease_expires_at > now()`,
		workerID, ttl.Seconds(),
	)
	return err
}

// DeleteRangesByJobID removes every range of a job
func (p *Postgres) DeleteRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var exists bool
	if err := p.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM jobs WHERE id = $1)`, jobID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrJobNotFoundByID
	}

	_, err := p.db.ExecContext(ctx, `DELETE FROM ranges WHERE job_id = $1`, jobID)
	return err
}
