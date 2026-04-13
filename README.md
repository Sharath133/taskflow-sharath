# TaskFlow (Backend)

## 1. Overview

TaskFlow is a small production-style task management API: users register and log in with JWT, own **projects**, and manage **tasks** inside those projects (status, priority, assignee, due dates). Users see projects they **own** or **have tasks in** as assignee **or** task creator (`created_by`). Owners control project updates and deletion. Task **creators** who are not assignees can still **delete** tasks they created (see service rules).

**Stack:** Go 1.21, Gin, PostgreSQL 15, `sqlx`, `golang-jwt`, bcrypt, `golang-migrate` (via Docker), structured logging with `log/slog`, graceful shutdown on SIGINT/SIGTERM.

**Layout:** Go code lives under [`backend/`](backend/) (monorepo-style); Docker Compose and docs stay at the repository root.

## 2. Architecture Decisions

- **Architecture note:** For layering rules, error → HTTP mapping, transactions, and observability (`X-Request-ID`, structured logs), see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- **Layered layout:** `handlers` → `service` → `repository` under `backend/internal/`. Domain types and errors live in `backend/internal/domain`.
- **No ORM auto-migrate:** Schema is owned by versioned SQL migrations in `backend/migrations` (`*.up.sql` / `*.down.sql`), applied by the `migrate` container before the API starts.
- **JSON shape:** Success responses follow the take-home **Appendix A** style — no `data` wrapper; auth returns `{ "token", "user" }`, `GET /projects` returns `{ "projects": [...] }`, `GET /projects/:id/tasks` returns `{ "tasks": [...] }` (with optional pagination fields). See [docs/API.md](docs/API.md).
- **JWT in environment:** `JWT_SECRET` is required at startup; nothing secret is compiled into the binary.
- **Pagination (optional):** If `page` or `limit` is omitted on list endpoints, responses use the Appendix list shape only. If either is present, the same object adds `total`, `page`, and `limit` (defaults: page `1`, limit `20`, max `100`).
- **Intentionally omitted:** Refresh tokens, OAuth, rate limiting (defer to reverse proxy), and a bundled React app (backend-only; use the Postman collection or `go test`).

## 3. Running Locally

Assume **Docker Desktop** (or Docker Engine + Compose plugin) is installed; Go is **not** required on the host for the default path.

```bash
git clone https://github.com/Sharath133/taskflow-sharath
cd taskflow-sharath
cp .env.example .env
docker compose up --build
```

- PostgreSQL becomes healthy, **migrations run to completion**, then the API listens on **http://localhost:8080** (REST API — there is no separate browser app in this repo).
- Health check: `GET http://localhost:8080/health` → `200`.

**Optional — Go on the host:** from `backend/`, copy env (e.g. `cp ../.env .env` or use `.env.host` per [docs/SETUP.md](docs/SETUP.md)), then `go run ./cmd/api` or `go test ./...`.

## 4. Running Migrations

Migrations run **automatically** when you start the stack: the `migrate` service executes `migrate up` against `DATABASE_URL` after Postgres is ready.

For manual `up` / `down`, point the [golang-migrate CLI](https://github.com/golang-migrate/migrate) at **`backend/migrations`**. See [docs/SETUP.md](docs/SETUP.md) for example commands.

Every migration file has matching **up** and **down** SQL.

## 5. Test Credentials

Seed data (from migration `000002_seed_data`) — use after first `docker compose up`:

| Field    | Value            |
|----------|------------------|
| Email    | `test@example.com` |
| Password | `password123`    |

The seed also creates one project and three tasks (`todo`, `in_progress`, `done`). Use **Login** in Postman and paste the returned JWT into the `token` collection variable.

## 6. API Reference

- **Full reference:** [docs/API.md](docs/API.md)
- **Postman collection:** [docs/postman/TaskFlow.postman_collection.json](docs/postman/TaskFlow.postman_collection.json)
- **cURL samples:** [examples/requests.sh](examples/requests.sh)
- **Environment / config:** [docs/SETUP.md](docs/SETUP.md)

**Base URL:** `http://localhost:8080`

### Quick example: token + authenticated request

Log in with the [seed user](#5-test-credentials), read **`token`** from the JSON body, and send `Authorization: Bearer <token>` on protected routes.

**Bash (curl + jq):**

```bash
BASE_URL="http://localhost:8080"
TOKEN="$(curl -sS -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' | jq -r '.token')"
curl -sS "${BASE_URL}/projects" -H "Authorization: Bearer ${TOKEN}"
```

**PowerShell:**

```powershell
$base = "http://localhost:8080"
$login = Invoke-RestMethod -Method Post -Uri "$base/auth/login" `
  -ContentType "application/json" `
  -Body (@{ email = "test@example.com"; password = "password123" } | ConvertTo-Json)
$token = $login.token
Invoke-RestMethod -Uri "$base/projects" -Headers @{ Authorization = "Bearer $token" }
```

**Errors (examples):**

- Validation: `400` — `{"error":"validation failed","fields":{"email":"is required"}}`
- Unauthenticated: `401` — `{"error":"unauthorized"}`
- Forbidden: `403` — `{"error":"forbidden"}`
- Not found: `404` — `{"error":"not found"}`

Successful **`DELETE`** responses use **204** with an empty body (no JSON).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register; returns JWT + user |
| POST | `/auth/login` | No | Login; returns JWT + user |
| GET | `/projects` | Bearer | List accessible projects (`?page=&limit=` optional) |
| POST | `/projects` | Bearer | Create project (owner = current user) |
| GET | `/projects/:id` | Bearer | Project + all tasks + `task_count` |
| GET | `/projects/:id/stats` | Bearer | Task counts by `status` and by `assignee` |
| PATCH | `/projects/:id` | Bearer | Update name/description (owner only) |
| DELETE | `/projects/:id` | Bearer | Delete project and tasks (owner only) |
| GET | `/projects/:id/tasks` | Bearer | List tasks; `?status=`, `?assignee=` (UUID), `?page=&limit=` |
| POST | `/projects/:id/tasks` | Bearer | Create task |
| PATCH | `/tasks/:id` | Bearer | Partial update task |
| DELETE | `/tasks/:id` | Bearer | Delete if project owner or task creator |
| GET | `/health` | No | Liveness |

**Automated tests (from repo root):**

```bash
make test
```

Or: `cd backend && go test ./...`

Integration tests (Docker / Testcontainers): `cd backend && go test -tags=integration -v ./tests/...`

## 7. What You’d Do With More Time

- **CI** running `make test` and integration tests on every push.
- **Refresh tokens** and shorter-lived access tokens; JWT key rotation.
- **Rate limiting** and audit logging on sensitive actions.
- **OpenAPI** generated from code or maintained beside the Postman collection.
- **E2E** smoke job after `docker compose up`.

---

## Repository layout (quick)

- `backend/cmd/api` — process entrypoint, signal handling  
- `backend/internal/` — router, handlers, service, repository, domain, middleware  
- `backend/migrations` — PostgreSQL schema and seed  
- `backend/pkg` — shared packages (e.g. database)  
- `docker-compose.yml` — Postgres, migrate, API  
- `docs/` — API reference, setup, Postman collection  
