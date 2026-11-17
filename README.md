# kinescope-scrapper

Distributed system that scans numeric identifier ranges on kinescope, finds available video material, collects its metadata and downloads the files allowed for saving.

## Components

- `coordinator/` - Go/Gin service that manages jobs, workers, videos and events, see `coordinator/README.md`
- `agent/` - standalone python parser that checks identifiers and downloads videos, runnable on any number of remote servers
- `frontend/` - React admin panel and public video catalog
- `nginx/` - TLS gateway serving the frontend build and proxying `/api` and S3 presigned requests
- `garage/` - config of the single-node Garage S3 server
- `docker-compose.yml` - Postgres, Garage S3, backend, frontend build and nginx gateway

## Quickstart

```bash
cp .env.example .env
docker compose up -d --build
```

Edit `.env` first: database credentials, S3 keys and rpc secret, JWT secret and the bootstrap administrator account from `ADMIN_USERNAME`, `ADMIN_EMAIL` and `ADMIN_PASSWORD`.

The gateway serves https on port 9000, the backend is reachable directly on port 8080. Migrations are applied by the backend on startup.

## Development

Coordinator in memory mode:

```bash
cd coordinator
JWT_SECRET=dev-secret JWT_ISSUER=coordinator go run ./cmd/coordinator -port 8080
```

Frontend with hot reload:

```bash
cd frontend
npm install
npm run dev
```

Agent:

```bash
cd agent
pip install -r requirements.txt
python main.py
```
