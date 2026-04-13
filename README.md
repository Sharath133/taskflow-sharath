# TaskFlow (Backend)

## 1. Overview

TaskFlow is a small production-style task management API: users register and log in with JWT, own **projects**, and manage **tasks** inside those projects (status, priority, assignee, due dates). Assignees or task creators can see relevant projects; owners control project updates and deletion.

**Stack:** Go 1.21, Gin, PostgreSQL 15, `sqlx`, `golang-jwt`, bcrypt, `golang-migrate` (via Docker), structured logging with `log/slog`, graceful shutdown on SIGINT/SIGTERM.

## 2. Architecture Decisions

- **Architecture note:** For layering rules, error → HTTP mapping, transactions, and observability (`X-Request-ID`, structured logs), see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- **Layered layout:** `handlers` (HTTP + validation mapping) → `service` (rules and orchestration) → `repository` (SQL). Domain types and errors live in `domain` so HTTP and persistence share one vocabulary.
- **No ORM auto-migrate:** Schema is owned by versioned SQL migrations (`migrations/*.up.sql` / `*.down.sql`) applied by the `migrate` container before the API starts. This keeps review, rollbacks, and production changes explicit.
- **Access model:** A user sees projects they **own**, are **assigned** to on at least one task, or **created** a task in (via `created_by`). Project detail and task mutations require that same access; only the owner may update/delete a project.
- **JWT in environment:** `JWT_SECRET` is required at startup; nothing secret is compiled into the binary.
- **Pagination (optional query):** If `page` or `limit` is omitted on list endpoints, responses stay a plain JSON array under `data` for backward compatibility. If either is present, `data` becomes an object with `items`, `total`, `page`, and `limit` (defaults: page `1`, limit `20`, max `100`).
- **Intentionally omitted:** Refresh tokens, OAuth, rate limiting (defer to reverse proxy), and a bundled React app (this repo is backend-only; use the Postman collection or `go test`).

## 3. Running Locally

Assume **Docker Desktop** (or Docker Engine + Compose plugin) is installed; Go is **not** required on the host.

```bash
git clone <your-repo-url> taskflow
cd taskflow
cp .env.example .env
docker compose up --build
```

- PostgreSQL becomes healthy, **migrations run to completion**, then the API listens on **http://localhost:8080**.
- Health check: `GET http://localhost:8080/health` → `200`.

## 4. Running Migrations

Migrations run **automatically** when you start the stack: the `migrate` service executes `migrate up` against `DATABASE_URL` after Postgres is ready. You do not need manual steps for a normal `docker compose up`.

For manual `up` / `down` against your `DATABASE_URL`, use the [golang-migrate CLI](https://github.com/golang-migrate/migrate) locally or run the `migrate/migrate` image with `-path` and `-database` pointing at this repo’s `./migrations` folder.

Every migration file has matching **up** and **down** SQL.

## 5. Test Credentials

Seed data (from migration `000002_seed_data`) — use after first `docker compose up`:

| Field    | Value            |
|----------|------------------|
| Email    | `test@example.com` |
| Password | `password123`    |

The seed also creates one project and three tasks (`todo`, `in_progress`, `done`). Use **Login** in Postman and paste the returned JWT into the `token` collection variable.

## 6. API Reference

- **Full reference:** [docs/API.md](docs/API.md) — schemas, status codes, and every endpoint in detail.
- **Postman collection:** [docs/postman/TaskFlow.postman_collection.json](docs/postman/TaskFlow.postman_collection.json) — import into Postman or Bruno; set the `base_url` collection variable, then run **Login** (or **Register**) so the `token` variable is filled for protected requests.
- **More cURL samples:** [examples/requests.sh](examples/requests.sh)
- **Environment / config:** [docs/SETUP.md](docs/SETUP.md)

**Base URL:** `http://localhost:8080` (or your deployment host).

### Quick example: token + authenticated request

Log in with the [seed user](#5-test-credentials), read `data.token`, and send `Authorization: Bearer <token>` on protected routes.

**Bash (curl + jq):**

```bash
BASE_URL="http://localhost:8080"
TOKEN="$(curl -sS -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' | jq -r '.data.token')"
curl -sS "${BASE_URL}/projects" -H "Authorization: Bearer ${TOKEN}"
```

**PowerShell:**

```powershell
$base = "http://localhost:8080"
$login = Invoke-RestMethod -Method Post -Uri "$base/auth/login" `
  -ContentType "application/json" `
  -Body (@{ email = "test@example.com"; password = "password123" } | ConvertTo-Json)
$token = $login.data.token
Invoke-RestMethod -Uri "$base/projects" -Headers @{ Authorization = "Bearer $token" }
```

**Success responses** wrap payloads as `{"data": ...}` with `Content-Type: application/json`.

**Errors (examples):**

- Validation: `400` — `{"error":"validation failed","fields":{"email":"is required"}}`
- Unauthenticated: `401` — `{"error":"unauthorized"}`
- Forbidden: `403` — `{"error":"forbidden"}`
- Not found: `404` — `{"error":"not found"}`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register; returns JWT + user |
| POST | `/auth/login` | No | Login; returns JWT + user |
| GET | `/projects` | Bearer | List accessible projects (`?page=&limit=` optional) |
| POST | `/projects` | Bearer | Create project (owner = current user) |
| GET | `/projects/:id` | Bearer | Project + all tasks + `task_count` |
| GET | `/projects/:id/stats` | Bearer | Task counts by `status` and by `assignee` (`__unassigned__` when no assignee) |
| PATCH | `/projects/:id` | Bearer | Update name/description (owner only) |
| DELETE | `/projects/:id` | Bearer | Delete project and tasks (owner only) |
| GET | `/projects/:id/tasks` | Bearer | List tasks; `?status=`, `?assignee=` (UUID), `?page=&limit=` |
| POST | `/projects/:id/tasks` | Bearer | Create task |
| PATCH | `/tasks/:id` | Bearer | Partial update task |
| DELETE | `/tasks/:id` | Bearer | Delete if project owner or task creator |
| GET | `/health` | No | Liveness |

**Automated tests:**

```bash
go test ./...
```

(Handler tests use `httptest` and stubbed services; no database required.)

## 7. What You’d Do With More Time

- **True DB integration tests** (pipeline with Testcontainers or a disposable Postgres) for full auth + task flows.
- **Refresh tokens** and shorter-lived access tokens; key rotation for JWT.
- **Rate limiting** and audit logging on sensitive actions.
- **OpenAPI** generated from code or maintained beside the Postman collection.
- **E2E** smoke job in CI after `docker compose up`.
- Tighten **list response** shape so paginated and non-paginated responses use one schema (breaking change, needs versioning).

---

## Repository layout (quick)

- `cmd/api` — process entrypoint, signal handling, server config  
- `internal/router` — routes and middleware  
- `internal/handlers`, `internal/service`, `internal/repository` — HTTP, business rules, SQL  
- `migrations` — PostgreSQL schema and seed  
- `docs/postman` — API collection  
