//go:build postgres && !sqlite

package storage

import (
	"context"
	"crypto/ed25519"

	"coordinator/internal/domain"
)

const workerColumns = `
	id, name, host, ssh_port, ssh_user, ssh_key, threads, storage_path,
	min_free_space_gb, status, public_key`

func scanWorker(s rowScanner) (domain.Worker, error) {
	var (
		w         domain.Worker
		publicKey []byte
	)
	err := s.Scan(
		&w.ID, &w.Config.Name, &w.Config.Host, &w.Config.SSHPort,
		&w.Config.SSHUser, &w.Config.SSHKey, &w.Config.Threads,
		&w.Config.StoragePath, &w.Config.MinFreeSpaceGB, &w.Status, &publicKey,
	)
	if err != nil {
		return domain.Worker{}, err
	}
	w.PublicKey = ed25519.PublicKey(publicKey)
	return w, nil
}

func (p *Postgres) GetWorkerByID(ctx context.Context, workerID int) (domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return domain.Worker{}, err
	}

	w, err := scanWorker(p.db.QueryRowContext(ctx, `SELECT `+workerColumns+` FROM workers WHERE id = $1`, workerID))
	return w, withNotFound(err, errWorkerNotFound)
}

func (p *Postgres) GetWorkers(ctx context.Context) ([]domain.Worker, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := p.db.QueryContext(ctx, `SELECT `+workerColumns+` FROM workers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workers := []domain.Worker{}
	for rows.Next() {
		w, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, rows.Err()
}

func (p *Postgres) AddWorker(ctx context.Context, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if worker.Status == "" {
		worker.Status = domain.WorkerStatusOff
	}
	if worker.PublicKey == nil {
		worker.PublicKey = []byte{}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if worker.ID == 0 {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO workers (
				name, host, ssh_port, ssh_user, ssh_key, threads, storage_path,
				min_free_space_gb, status, public_key
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			worker.Config.Name, worker.Config.Host, worker.Config.SSHPort,
			worker.Config.SSHUser, worker.Config.SSHKey, worker.Config.Threads,
			worker.Config.StoragePath, worker.Config.MinFreeSpaceGB,
			worker.Status, worker.PublicKey,
		)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO workers (
				id, name, host, ssh_port, ssh_user, ssh_key, threads, storage_path,
				min_free_space_gb, status, public_key
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name, host = EXCLUDED.host, ssh_port = EXCLUDED.ssh_port,
				ssh_user = EXCLUDED.ssh_user, ssh_key = EXCLUDED.ssh_key,
				threads = EXCLUDED.threads, storage_path = EXCLUDED.storage_path,
				min_free_space_gb = EXCLUDED.min_free_space_gb,
				status = EXCLUDED.status, public_key = EXCLUDED.public_key`,
			worker.ID, worker.Config.Name, worker.Config.Host, worker.Config.SSHPort,
			worker.Config.SSHUser, worker.Config.SSHKey, worker.Config.Threads,
			worker.Config.StoragePath, worker.Config.MinFreeSpaceGB,
			worker.Status, worker.PublicKey,
		)
		if err == nil {
			err = syncSequence(ctx, tx, "workers")
		}
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *Postgres) ReplaceWorker(ctx context.Context, worker domain.Worker) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if worker.PublicKey == nil {
		worker.PublicKey = []byte{}
	}

	result, err := p.db.ExecContext(ctx, `
		UPDATE workers SET
			name = $2, host = $3, ssh_port = $4, ssh_user = $5, ssh_key = $6,
			threads = $7, storage_path = $8, min_free_space_gb = $9,
			status = $10, public_key = $11
		WHERE id = $1`,
		worker.ID, worker.Config.Name, worker.Config.Host, worker.Config.SSHPort,
		worker.Config.SSHUser, worker.Config.SSHKey, worker.Config.Threads,
		worker.Config.StoragePath, worker.Config.MinFreeSpaceGB,
		worker.Status, worker.PublicKey,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errWorkerNotFound
	}
	return nil
}

func (p *Postgres) DeleteWorkerByID(ctx context.Context, workerID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := p.db.ExecContext(ctx, `DELETE FROM workers WHERE id = $1`, workerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errWorkerNotFound
	}
	return nil
}
