# TaskFlow — Setup guide

Step-by-step instructions to run the TaskFlow API locally with Docker, run database migrations, and verify the service.

## Table of contents

- [Prerequisites](#prerequisites)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Migrations](#migrations)
- [Accessing the API](#accessing-the-api)
- [Test credentials](#test-credentials)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

| Tool | Purpose |
|------|---------|
| **Docker** (Docker Desktop on Windows, or Docker Engine + Compose plugin) | Runs PostgreSQL, migration job, and API container |
| **Git** | Clone the repository |

Optional:

- **Go 1.22+** — if you run the API or tests outside Docker
- **golang-migrate CLI** — manual migration `up` / `down` against a custom `DATABASE_URL`

---

## Quick start

1. **Clone the repository**

   ```bash
   git clone <your-repo-url> taskflow
   cd taskflow
   ```

2. **Create environment file**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` if you need to change secrets, ports, or database names. For the default Compose stack, the values in `.env.example` work as-is.

3. **Start the stack**

   ```bash
   docker compose up --build
   ```

   Order of operations:

   - PostgreSQL starts and becomes **healthy**.
   - The **migrate** service runs `migrate up` once successfully.
   - The **api** service starts and listens on port **8080** (mapped from the container).

4. **Verify**

   ```bash
   curl -s http://localhost:8080/health
   ```

   Expected:

   ```json
   {"status":"ok"}
   ```

---

## Configuration

Key variables in `.env` (see `.env.example` for the full list):

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string (Compose uses host `postgres`) |
| `JWT_SECRET` | Secret for signing JWTs (change in production) |
| `JWT_EXPIRY` | Token lifetime (e.g. `24h`) |
| `SERVER_PORT` | HTTP port inside the API container (default `8080`) |
| `BCRYPT_COST` | bcrypt cost; must be **≥ 12** |
| `GIN_MODE` | `release` or `debug` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Initialize the Postgres container; must stay consistent with `DATABASE_URL` |

`CORS_ALLOWED_ORIGINS` can be set for browser clients (see `backend/internal/router`); it is optional for local API-only use.

---

## Migrations

### Automatic (recommended)

When you run `docker compose up`, the `migrate` service applies all SQL files under `./backend/migrations` after Postgres is ready. You do **not** need a separate migration command for normal development.

### Manual

If you need to run migrations yourself against a database:

1. Install [golang-migrate](https://github.com/golang-migrate/migrate).
2. Point `-path` at this repo’s `backend/migrations` folder and `-database` at your `DATABASE_URL`.

Example (URL-encode passwords if they contain special characters):

```bash
migrate -path ./backend/migrations -database "$DATABASE_URL" up
migrate -path ./backend/migrations -database "$DATABASE_URL" down 1
```

Every migration has matching **up** and **down** SQL.

---

## Accessing the API

| Item | Value |
|------|--------|
| **Base URL** | `http://localhost:8080` |
| **Health** | `GET http://localhost:8080/health` |
| **Docs** | [API.md](./API.md) — full endpoint reference |

**Postman:** import [docs/postman/TaskFlow.postman_collection.json](./postman/TaskFlow.postman_collection.json). Set collection variables `base_url` and `token` (token is filled automatically after **Login** or **Register** if you use the collection’s test scripts).

**cURL:** see [examples/requests.sh](../examples/requests.sh).

---

## Test credentials

After migrations (including seed `000002_seed_data`), a demo user and data exist:

| Field | Value |
|-------|--------|
| **Email** | `test@example.com` |
| **Password** | `password123` |

**Deterministic seed UUIDs** (useful for scripts and demos):

| Resource | UUID |
|----------|------|
| Seed user | `10000000-0000-4000-8000-000000000001` |
| Seed project | `20000000-0000-4000-8000-000000000001` |
| Seed task (todo) | `30000000-0000-4000-8000-000000000001` |
| Seed task (in progress) | `30000000-0000-4000-8000-000000000002` |
| Seed task (done) | `30000000-0000-4000-8000-000000000003` |

---

## Troubleshooting

### Port 8080 already in use

Another process is bound to `8080`. Either stop that process or change the host mapping in `docker-compose.yml` (e.g. `"8081:8080"`) and call `http://localhost:8081`.

### API exits or cannot connect to database

- Ensure `postgres` is healthy: `docker compose ps`.
- Confirm `DATABASE_URL` in `.env` uses host `postgres` when running **inside** Compose (not `localhost`).
- If you run the Go binary on the host machine, use `localhost` and the published Postgres port instead.

### Migrations fail or `migrate` exits non-zero

- Check logs: `docker compose logs migrate`.
- Ensure `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB` match `DATABASE_URL`.
- If the volume has a corrupted state, you can reset **only for local dev**: `docker compose down -v` (this **deletes** database data), then `docker compose up --build` again.

### `JWT_SECRET` / bcrypt errors at startup

- `BCRYPT_COST` must be at least **12** (validated in config).
- `JWT_SECRET` must be non-empty for token issuance and validation.

### Login returns `401` with `unauthorized`

- Wrong or unknown email/password responds with `{"error":"unauthorized"}` (same message as missing/invalid JWT) to avoid leaking whether an email exists.
- Use the seed user `test@example.com` / `password123` after a fresh migrate with seed.
- Emails are normalized to lowercase; extra spaces are trimmed on register/login.

### 401 on all protected routes

- Send `Authorization: Bearer <token>` (note the space after `Bearer`).
- Token may be expired; log in again. Expiry is controlled by `JWT_EXPIRY`.

### CORS errors in the browser

- Set `CORS_ALLOWED_ORIGINS` to a comma-separated list of allowed origins, or configure a reverse proxy. Empty origins may default to permissive behavior without credentials; see `backend/internal/router` for details.

---

## Next steps

- Read [API.md](./API.md) for request/response formats and error codes.
- Run automated tests: `go test ./...` (from repo root, with Go installed).
