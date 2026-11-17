# Coordinator

Go/Gin service that coordinates scraping workers.

## Run

```bash
cd coordinator
go run ./cmd/coordinator -port 8080
```

Required environment for the default build:

- `JWT_SECRET` - non-empty secret used to sign JWTs
- `JWT_ISSUER` - non-empty token issuer

Optional:
- `LOG_LEVEL` - log verbosity: debug, info, warn or error, defaults to info
- `LOG_FORMAT` - set to json for structured JSON logs, text otherwise
- `ADMIN_USERNAME` - on first start an administrator account is created when this, `ADMIN_EMAIL` and `ADMIN_PASSWORD` are set
- `ADMIN_EMAIL` - email of the bootstrap administrator
- `ADMIN_PASSWORD` - password of the bootstrap administrator
- `ARGON2_MEMORY` - Argon2id memory cost in KiB, defaults to 65536
- `ARGON2_ITERATIONS` - Argon2id time cost, defaults to 3
- `ARGON2_THREADS` - Argon2id parallelism, defaults to 2
- `PASSWORD_HASH_LEN` - password hash length in bytes, defaults to 32
- `SALT_SIZE` - password salt size in bytes, defaults to 16
- `S3_SERVER_URL` - S3 endpoint used to presign upload and download links
- `S3_ACCESS_KEY` - S3 access key
- `S3_SECRET_KEY` - S3 secret key
- `S3_REGION` - S3 region
- `S3_BUCKET` - bucket holding downloaded videos

## Storage

The default build uses an in-memory store and loses all data on restart. PostgreSQL backend is available using the `postgres` build tag:

```bash
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/coordinator?sslmode=disable' \
JWT_SECRET=dev-secret JWT_ISSUER=coordinator \
go run -tags postgres ./cmd/coordinator
```

`POSTGRES_DSN` is required by the Postgres build. Embedded migrations in `internal/storage/migrations/` are applied automatically on startup and recorded in `schema_migrations`.

## Tests

```bash
go build ./... && go vet ./...
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/coordinator_test?sslmode=disable' \
  go test -tags postgres ./internal/storage
```

The storage integration test truncates the target database, so point it at a disposable database only.

## Job ranges

A job processes an inclusive range of numeric identifiers. A range with `to = 0` is open-ended and keeps working without an upper bound, workers are always leased bounded blocks of at most 1000 identifiers carved from the job range on demand. Open jobs are never completed automatically, stop them by hand:

```json
{"range": {"from": 0, "to": 0}}
```

An empty `to` in the create form means the same thing. Finite ranges are completed automatically when every identifier is processed.

## API

The HTTP API is described by `openapi.yaml` next to this file, a rendered copy lives in `openapi.html`. Live updates for jobs, workers, metrics and events are pushed over server-sent events on `/stream/*`, logs use the structured `slog` format configured by `LOG_LEVEL` and `LOG_FORMAT`.
