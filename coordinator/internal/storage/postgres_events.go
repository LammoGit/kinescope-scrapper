//go:build postgres && !sqlite

package storage

import (
	"context"
	"time"

	"coordinator/internal/domain"
)

const eventColumns = `
	occurred_at, level, component, job_id, worker_id, range_id,
	resource_id, operation, result, error_code, message`

func scanEvent(s rowScanner) (domain.Event, error) {
	var e domain.Event
	err := s.Scan(
		&e.Time, &e.Level, &e.Component, &e.JobID, &e.WorkerID, &e.RangeID,
		&e.ResourceID, &e.Operation, &e.Result, &e.ErrorCode, &e.Message,
	)
	return e, err
}

func (p *Postgres) AddEvent(ctx context.Context, event domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO events (
			occurred_at, level, component, job_id, worker_id, range_id,
			resource_id, operation, result, error_code, message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		event.Time, event.Level, event.Component, event.JobID, event.WorkerID,
		event.RangeID, event.ResourceID, event.Operation, event.Result,
		event.ErrorCode, event.Message,
	)
	return err
}

func (p *Postgres) GetEvents(ctx context.Context) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.queryEvents(ctx, `SELECT `+eventColumns+` FROM events ORDER BY id`)
}

func (p *Postgres) GetEventByID(ctx context.Context, id int) (domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return domain.Event{}, err
	}

	event, err := scanEvent(p.db.QueryRowContext(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id))
	return event, withNotFound(err, errEventNotFound)
}

func (p *Postgres) GetEventsByJobID(ctx context.Context, jobID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.queryEvents(ctx,
		`SELECT `+eventColumns+` FROM events WHERE job_id = $1 ORDER BY id`,
		jobID,
	)
}

func (p *Postgres) GetEventsByWorkerID(ctx context.Context, workerID int) ([]domain.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.queryEvents(ctx,
		`SELECT `+eventColumns+` FROM events WHERE worker_id = $1 ORDER BY id`,
		workerID,
	)
}

func (p *Postgres) queryEvents(ctx context.Context, query string, args ...any) ([]domain.Event, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []domain.Event{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
