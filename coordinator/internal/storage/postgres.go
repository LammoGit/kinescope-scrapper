//go:build postgres && !sqlite

package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"coordinator/internal/repos"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Postgres is a durable Storage implementation backed by PostgreSQL.
type Postgres struct {
	db *sql.DB
}

var _ Storage = (*Postgres)(nil)

// NewStorage opens the database from DATABASE_URL, verifies the connection and
// applies pending embedded migrations.
func NewStorage() (Storage, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("storage: DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open postgres: %w", err)
	}

	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: ping postgres: %w", err)
	}

	p := &Postgres{db: db}
	if err := p.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return p, nil
}

// Close releases the underlying connection pool.
func (p *Postgres) Close() error {
	return p.db.Close()
}

//go:embed migrations/*.sql
var migrationFS embed.FS

const migrationLockKey int64 = 0x6b696e6573636f70 // "kinescop"

// migrate applies every embedded migration exactly once. An advisory lock
// serializes concurrent coordinator instances during startup.
func (p *Postgres) migrate(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text        PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("storage: create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("storage: list migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		version := strings.TrimSuffix(path.Base(file), ".sql")

		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("storage: begin migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockKey); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: lock migration %s: %w", version, err)
		}

		var applied bool
		if err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`,
			version,
		).Scan(&applied); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: check migration %s: %w", version, err)
		}
		if applied {
			_ = tx.Rollback()
			continue
		}

		body, err := migrationFS.ReadFile(file)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: read migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: apply migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("storage: record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("storage: commit migration %s: %w", version, err)
		}
	}

	return nil
}

// syncSequence moves the identity sequence of table past its current maximum.
// It is needed only after inserting a row with an explicit identifier.
func syncSequence(ctx context.Context, tx *sql.Tx, table string) error {
	_, err := tx.ExecContext(ctx,
		fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE((SELECT MAX(id) FROM %s), 1))`,
			table, table,
		),
	)
	return err
}

// jsonBytes marshals a value for a JSONB column, storing JSON null as [].
func jsonBytes(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if string(b) == "null" {
		return []byte("[]"), nil
	}
	return b, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func withNotFound(err, target error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return target
	}
	return err
}

// Repository errors are reused so HTTP status mapping stays identical to the
// in-memory implementation.
var (
	errJobNotFound    = repos.ErrJobNotFoundByID
	errRangeNotFound  = repos.ErrRangeNotFoundByID
	errWorkerNotFound = repos.ErrWorkerNotFoundByID
	errVideoNotFound  = repos.ErrVideoNotFoundByID
	errEventNotFound  = repos.ErrEventNotFoundByID
	errUserNotFound   = repos.ErrUserNotFoundByID
)
