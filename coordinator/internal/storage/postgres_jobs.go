//go:build postgres && !sqlite

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"coordinator/internal/domain"
	"coordinator/internal/repos"
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

// CreateJob creates a job and returns it. When both name and config.Name are
// empty, the job is named job-<id>, matching the in-memory storage.
func (p *Postgres) CreateJob(ctx context.Context, name string, config domain.JobConfig) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	if name == "" {
		name = config.Name
	}
	servers, err := jsonBytes(config.Servers)
	if err != nil {
		return domain.Job{}, err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Job{}, err
	}
	defer tx.Rollback()

	var id int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO jobs (
			name, range_from, range_to, request_timeout, max_tries,
			global_rate_limit_rps, stop_on_consecutive_missing, status,
			paused, workers, servers
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, '[]'::jsonb, $9)
		RETURNING id`,
		name, config.Range.From, config.Range.To, config.RequestTimeout,
		config.MaxTries, config.GlobalRateLimitRPS, config.StopOnConsecutiveMissing,
		domain.JobStatusCreated, servers,
	).Scan(&id)
	if err != nil {
		return domain.Job{}, err
	}

	if name == "" {
		name = fmt.Sprintf("job-%d", id)
		if _, err := tx.ExecContext(ctx, `UPDATE jobs SET name = $1 WHERE id = $2`, name, id); err != nil {
			return domain.Job{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Job{}, err
	}

	config.Name = name
	return domain.Job{
		ID:      id,
		Config:  config,
		Workers: []int{},
		Status:  domain.JobStatusCreated,
	}, nil
}

// AssignJob adds a worker to the job without duplicating it.
func (p *Postgres) AssignJob(ctx context.Context, id int, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT workers FROM jobs WHERE id = $1 FOR UPDATE`, id).Scan(&raw); err != nil {
		return withNotFound(err, errJobNotFound)
	}

	workers := []int{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &workers); err != nil {
			return fmt.Errorf("decode job workers: %w", err)
		}
	}
	for _, current := range workers {
		if current == worker.ID {
			return tx.Commit()
		}
	}
	workers = append(workers, worker.ID)

	encoded, err := json.Marshal(workers)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET workers = $1, updated_at = now() WHERE id = $2`, encoded, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Postgres) GetJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	job, err := scanJob(p.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id))
	return job, withNotFound(err, errJobNotFound)
}

func (p *Postgres) GetJobs(ctx context.Context) ([]domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx, `SELECT `+jobColumns+` FROM jobs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (p *Postgres) DeleteJob(ctx context.Context, id int) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	job, err := scanJob(p.db.QueryRowContext(ctx, `DELETE FROM jobs WHERE id = $1 RETURNING `+jobColumns, id))
	return job, withNotFound(err, errJobNotFound)
}

func (p *Postgres) setJobState(ctx context.Context, id int, status string, paused bool, deassign bool) (domain.Job, error) {
	if err := ctx.Err(); err != nil {
		return domain.Job{}, err
	}

	query := `
		UPDATE jobs
		SET status = $2, paused = $3, updated_at = now()
		WHERE id = $1
		RETURNING ` + jobColumns
	if deassign {
		query = `
			UPDATE jobs
			SET status = $2, paused = $3, workers = '[]'::jsonb, updated_at = now()
			WHERE id = $1
			RETURNING ` + jobColumns
	}

	job, err := scanJob(p.db.QueryRowContext(ctx, query, id, status, paused))
	return job, withNotFound(err, errJobNotFound)
}

func (p *Postgres) StartJob(ctx context.Context, id int) (domain.Job, error) {
	return p.setJobState(ctx, id, domain.JobStatusRunning, false, false)
}

func (p *Postgres) PauseJob(ctx context.Context, id int) (domain.Job, error) {
	return p.setJobState(ctx, id, domain.JobStatusPaused, true, false)
}

func (p *Postgres) ResumeJob(ctx context.Context, id int) (domain.Job, error) {
	return p.setJobState(ctx, id, domain.JobStatusRunning, false, false)
}

func (p *Postgres) StopJob(ctx context.Context, id int) (domain.Job, error) {
	return p.setJobState(ctx, id, domain.JobStatusStopped, true, true)
}

func (p *Postgres) ReplaceJob(ctx context.Context, job domain.Job) error {
	if err := ctx.Err(); err != nil {
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
		job.Expiration, workers, servers,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errJobNotFound
	}
	return nil
}

func (p *Postgres) AddRange(ctx context.Context, rangeBlock domain.Range) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if rangeBlock.ID == 0 {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO ranges (
				job_id, range_start, range_end, next_id, status, leased_by,
				lease_expires_at, attempt, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			rangeBlock.JobID, rangeBlock.RangeStart, rangeBlock.RangeEnd,
			rangeBlock.NextID, rangeBlock.Status, rangeBlock.LeasedBy,
			nullableTime(rangeBlock.LeaseExpiresAt), rangeBlock.Attempt,
			rangeBlock.CreatedAt, rangeBlock.UpdatedAt,
		)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO ranges (
				id, job_id, range_start, range_end, next_id, status, leased_by,
				lease_expires_at, attempt, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				job_id = EXCLUDED.job_id, range_start = EXCLUDED.range_start,
				range_end = EXCLUDED.range_end, next_id = EXCLUDED.next_id,
				status = EXCLUDED.status, leased_by = EXCLUDED.leased_by,
				lease_expires_at = EXCLUDED.lease_expires_at, attempt = EXCLUDED.attempt,
				created_at = EXCLUDED.created_at, updated_at = EXCLUDED.updated_at`,
			rangeBlock.ID, rangeBlock.JobID, rangeBlock.RangeStart, rangeBlock.RangeEnd,
			rangeBlock.NextID, rangeBlock.Status, rangeBlock.LeasedBy,
			nullableTime(rangeBlock.LeaseExpiresAt), rangeBlock.Attempt,
			rangeBlock.CreatedAt, rangeBlock.UpdatedAt,
		)
		if err == nil {
			err = syncSequence(ctx, tx, "ranges")
		}
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Postgres) GetRangeByID(ctx context.Context, id int) (domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return domain.Range{}, err
	}

	r, err := scanRange(p.db.QueryRowContext(ctx, `SELECT `+rangeColumns+` FROM ranges WHERE id = $1`, id))
	return r, withNotFound(err, errRangeNotFound)
}

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

func (p *Postgres) GetUnassignedRanges(ctx context.Context) ([]domain.Range, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx, `
		SELECT `+rangeColumns+`
		FROM ranges
		WHERE status NOT IN ('completed', 'stopped', 'failed')
		  AND (leased_by = 0 OR lease_expires_at IS NULL OR lease_expires_at <= now())
		ORDER BY job_id, range_start, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return collectRanges(rows)
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

// TryClaimRange atomically leases an unassigned or expired range.
func (p *Postgres) TryClaimRange(ctx context.Context, rangeID, workerID int, leaseTTL time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var claimed int
	err := p.db.QueryRowContext(ctx, `
		UPDATE ranges
		SET leased_by = $2,
		    lease_expires_at = now() + make_interval(secs => $3),
		    status = 'running',
		    updated_at = now()
		WHERE id = $1
		  AND status NOT IN ('completed', 'stopped', 'failed')
		  AND (leased_by = 0 OR lease_expires_at IS NULL OR lease_expires_at <= now())
		RETURNING id`,
		rangeID, workerID, leaseTTL.Seconds(),
	).Scan(&claimed)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	var (
		status string
		leased int
		lease  sql.NullTime
	)
	err = p.db.QueryRowContext(ctx,
		`SELECT status, leased_by, lease_expires_at FROM ranges WHERE id = $1`,
		rangeID,
	).Scan(&status, &leased, &lease)
	if err != nil {
		return withNotFound(err, errRangeNotFound)
	}
	if status == domain.JobStatusCompleted || status == domain.JobStatusStopped || status == domain.JobStatusFailed {
		return repos.ErrRangeIsCompleted
	}
	if leased != 0 && (!lease.Valid || time.Now().Before(lease.Time)) {
		return repos.ErrRangeIsTaken
	}
	return repos.ErrRangeIsTaken
}

func (p *Postgres) RenewLeases(ctx context.Context, workerID int, leaseTTL time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE ranges
		SET lease_expires_at = now() + make_interval(secs => $2), updated_at = now()
		WHERE leased_by = $1 AND status = 'running'`,
		workerID, leaseTTL.Seconds(),
	)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	return int(affected), err
}

func (p *Postgres) ReleaseRangeByID(ctx context.Context, rangeID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE ranges
		SET leased_by = 0,
		    lease_expires_at = NULL,
		    status = CASE WHEN next_id > range_end THEN 'completed' ELSE 'created' END,
		    updated_at = now()
		WHERE id = $1`,
		rangeID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errRangeNotFound
	}
	return nil
}

func (p *Postgres) ReleaseRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var exists bool
	if err := p.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM jobs WHERE id = $1)`, jobID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errJobNotFound
	}

	_, err := p.db.ExecContext(ctx, `
		UPDATE ranges
		SET leased_by = 0,
		    lease_expires_at = NULL,
		    status = CASE WHEN next_id > range_end THEN 'completed' ELSE 'created' END,
		    updated_at = now()
		WHERE job_id = $1`,
		jobID,
	)
	return err
}

func (p *Postgres) CompleteRange(ctx context.Context, rangeID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE ranges
		SET leased_by = 0, lease_expires_at = NULL, status = 'completed', updated_at = now()
		WHERE id = $1`,
		rangeID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errRangeNotFound
	}
	return nil
}

func (p *Postgres) DeleteRangesByJobID(ctx context.Context, jobID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err := p.db.ExecContext(ctx, `DELETE FROM ranges WHERE job_id = $1`, jobID)
	return err
}
