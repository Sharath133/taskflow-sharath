# TaskFlow — architecture

This document describes how the backend is structured, how errors become HTTP responses, when transactions are used, and what was intentionally left out. It complements [README.md](../README.md).

## Layering and call direction

| Layer | Package(s) | Responsibility |
|--------|------------|------------------|
| **HTTP** | `backend/internal/handlers`, `backend/internal/router`, `backend/internal/middleware` | Routing, JSON, binding, status codes, auth middleware |
| **Application** | `backend/internal/service` | Access rules, validation orchestration, use-case flow |
| **Persistence** | `backend/internal/repository` | SQL via `sqlx`; map rows to `backend/internal/domain` types |
| **Domain** | `backend/internal/domain` | Shared models and sentinel errors (no SQL, no Gin) |
| **Infrastructure** | `backend/internal/config`, `backend/internal/auth`, `backend/pkg/database`, `backend/internal/observability` | Config, JWT/password helpers, DB connection, request correlation |

**Allowed dependencies (who may call whom):**

- Handlers call **only** services (and handler helpers). They do not import repositories.
- Services call **repositories** and **domain**; they do not import Gin or write HTTP responses.
- Repositories call **domain** and the database; they do not implement business rules (ownership, “can this user see this project?”).
- Middleware sits on the HTTP edge; it may use `backend/internal/observability` for request-scoped context.

## Errors → HTTP

Services return `error` values built from `backend/internal/domain`:

| Error / type | Typical HTTP | Response shape |
|----------------|-------------|----------------|
| `*domain.ValidationError` | 400 | `{"error":"validation failed","fields":{...}}` |
| `errors.Is(err, domain.ErrNotFound)` | 404 | `{"error":"not found"}` |
| `errors.Is(err, domain.ErrForbidden)` | 403 | `{"error":"forbidden"}` |
| `errors.Is(err, domain.ErrUnauthorized)` (after JWT; insufficient access) | 403 | `{"error":"forbidden"}` (see note) |
| `errors.Is(err, domain.ErrConflict)` | 409 | Handler-specific message |
| Other / wrapped errors | 500 | Generic message; details in logs only |

**Note:** For authenticated routes, “not allowed to do this on this resource” is mapped to **403** to distinguish it from **401** (missing/invalid JWT), which the auth middleware handles before the handler runs.

Handlers use `handlers.HandleServiceError` where applicable, then fall back to 500 for unexpected errors.

## Transactions

Repositories accept a `sqlx` connection interface implemented by both `*sqlx.DB` and `*sqlx.Tx`, so the same code paths can run inside a transaction.

**When a transaction is used today**

- **Task update** (`TaskService.Update`): `UPDATE` then reload by ID run in **one** transaction so the returned row always matches the committed update (no half-applied state if the reload failed mid-flight).
- **Project update** (`ProjectService.Update`): same pattern for `UPDATE` + `FindByID`.

**When a transaction is not used**

- Single-statement operations (e.g. create user, create task, delete project with FK cascade) are already atomic in PostgreSQL.
- **Register** relies on a unique constraint on email for races between “check” and “insert”; a transaction alone would not remove that race without stricter isolation, which is unnecessary given the DB constraint.
- **List + count** pagination stays as separate queries; a snapshot across both is not a stated product requirement.

Helper: `repository.WithTx(ctx, db, fn)` builds transactional `Repositories` and commits or rolls back once.

## Observability

- **`X-Request-ID`**: Accepted from the client when present; otherwise generated (UUID). Echoed on the response and attached to the request `context` (`backend/internal/observability`) for downstream use.
- **Access logs**: Structured JSON via `log/slog` with `request_id`, `method`, `path`, `status`, `duration_ms`, `client_ip`, `user_agent` (truncated), and optional Gin error string.
- **Panics**: Recovery middleware logs `request_id`, stack trace, and returns a generic JSON 500.

## Intentional omissions (see README)

Refresh tokens, OAuth, in-process rate limiting (defer to reverse proxy), and a bundled frontend remain out of scope for this repository unless requirements change.
