# Coordinator

Go/Gin service that coordinates scraping workers.

## Run

```bash
cd coordinator
go run ./cmd/coordinator -port 8080
```

Required environment for the default build:

- `JWT_SECRET` — non-empty secret used to sign JWTs
- `JWT_ISSUER` — non-empty token issuer

Optional: `ARGON2_MEMORY`, `ARGON2_ITERATIONS`, `ARGON2_THREADS`,
`PASSWORD_HASH_LEN`, `SALT_SIZE`, `S3_SERVER_URL`.

## Storage

The default build uses an in-memory store and loses all data on restart. A
PostgreSQL backend is available behind the `postgres` build tag:

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/coordinator?sslmode=disable' \
JWT_SECRET=dev-secret JWT_ISSUER=coordinator \
go run -tags postgres ./cmd/coordinator
```

`DATABASE_URL` is required by the Postgres build. Embedded migrations in
`internal/storage/migrations/` are applied automatically on startup and recorded
in `schema_migrations`.

The `sqlite` tag is reserved but not implemented; `postgres` and `sqlite` are
mutually exclusive.

## Tests

```bash
go build ./... && go vet ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/coordinator_test?sslmode=disable' \
  go test -tags postgres ./internal/storage
```

The storage integration test truncates the target database, so point it at a
disposable database only.
