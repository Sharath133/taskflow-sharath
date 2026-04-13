# TaskFlow API Reference

REST API for projects and tasks with JWT authentication. All JSON responses use `Content-Type: application/json` unless noted.

## Table of contents

- [Base URL](#base-url)
- [Conventions](#conventions)
- [Authentication](#authentication)
- [Error handling](#error-handling)
- [Endpoints](#endpoints)
  - [Health](#get-health)
  - [Auth](#auth)
  - [Projects](#projects)
  - [Tasks](#tasks)
- [Enumerations](#enumerations)

---

## Base URL

| Environment | URL |
|-------------|-----|
| Local (Docker Compose default) | `http://localhost:8080` |
| Custom deployment | `https://<your-host>` (no path prefix; routes are at server root) |

There is **no** `/api/v1` prefix in the current server build.

---

## Conventions

- **Success payloads** match the take-home Appendix A shape: **no `data` envelope** — e.g. `{ "token", "user" }` for auth, `{ "projects": [...] }` for `GET /projects`, `{ "tasks": [...] }` for task lists, and a bare project or task object for create/update.

- **Exceptions:** Successful `DELETE` returns **204 No Content** with an empty body (no JSON).

- **Dates and times** in JSON use RFC 3339 / ISO-8601 (e.g. `"2026-04-12T10:30:00Z"`).

- **UUIDs** are standard UUID strings in paths and JSON.

---

## Authentication

TaskFlow uses **JWT access tokens** signed with **HS256**. After `POST /auth/register` or `POST /auth/login`, the response body includes `token` at the top level.

### Using the token

Send the token on every protected request:

| Header | Value |
|--------|--------|
| `Authorization` | `Bearer <your_jwt_here>` |

Example:

```http
GET /projects HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Token contents (claims)

The JWT payload includes (among standard `exp` / `iat` claims):

| Claim | Description |
|-------|-------------|
| `user_id` | User UUID as string |
| `email` | User email |

Token lifetime is configured by `JWT_EXPIRY` (see [SETUP.md](./SETUP.md)).

### When authentication fails

Missing header, wrong scheme, invalid signature, expired token, or malformed JWT all yield **401** with the same body (to avoid leaking details):

```json
{
  "error": "unauthorized"
}
```

---

## Error handling

### Validation (`400`)

```json
{
  "error": "validation failed",
  "fields": {
    "email": "must be a valid email",
    "password": "must be at least 8"
  }
}
```

Other `400` shapes:

- Invalid JSON body: `{"error":"invalid request body"}` (no `fields`)
- Invalid path UUID: `{"error":"invalid path parameter","fields":{"id":"must be a valid UUID"}}`
- Invalid query (tasks list): `{"error":"invalid query parameters"}` or validation-style `fields` for `status` / `assignee`
- Pagination: `{"error":"validation failed","fields":{"page":"must be a positive integer"}}` (or `limit`)

### Unauthorized (`401`)

- Protected route without valid Bearer token: `{"error":"unauthorized"}`
- Login with wrong or unknown credentials: `{"error":"unauthorized"}`

### Forbidden (`403`)

Authenticated user lacks permission (e.g. not project owner for update/delete):

```json
{
  "error": "forbidden"
}
```

### Not found (`404`)

```json
{
  "error": "not found"
}
```

### Conflict (`409`)

Registration with an email that already exists:

```json
{
  "error": "email already registered"
}
```

### Server error (`500`)

```json
{
  "error": "internal server error"
}
```

---

## Endpoints

### `GET /health`

Liveness check. **No authentication.**

| | |
|--|--|
| **Description** | Returns service health for load balancers and monitors. |
| **Request headers** | None required. |
| **Request body** | None. |

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Service is up. |

**Response body (200)** — note: **no** `data` wrapper.

```json
{
  "status": "ok"
}
```

---

## Auth

### `POST /auth/register`

Create a new user and return a JWT plus user profile. **No authentication.**

| | |
|--|--|
| **Description** | Registers a user; password is stored as a bcrypt hash. |
| **Request headers** | `Content-Type: application/json` |
| **Request body** | See below. |

**Request body (JSON)**

```json
{
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "password": "password123"
}
```

| Field | Type | Required | Rules |
|-------|------|----------|--------|
| `name` | string | Yes | 1–255 characters |
| `email` | string | Yes | Valid email, max 255 |
| `password` | string | Yes | 8–72 characters |

**Response codes**

| Code | Description |
|------|-------------|
| 201 | User created; JWT issued. |
| 400 | Validation error. |
| 409 | Email already registered. |
| 500 | Internal error. |

**Response body (201)**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "created_at": "2026-04-12T12:00:00Z"
  }
}
```

**Error example (409)**

```json
{
  "error": "email already registered"
}
```

**Error example (400)**

```json
{
  "error": "validation failed",
  "fields": {
    "email": "must be a valid email"
  }
}
```

---

### `POST /auth/login`

Exchange email and password for a JWT. **No authentication.**

| | |
|--|--|
| **Description** | Validates credentials and returns the same token + user shape as register. |
| **Request headers** | `Content-Type: application/json` |

**Request body (JSON)**

```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

| Field | Type | Required |
|-------|------|----------|
| `email` | string | Yes |
| `password` | string | Yes |

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Login successful. |
| 400 | Validation error. |
| 401 | Invalid email or password. |
| 500 | Internal error. |

**Response body (200)**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "10000000-0000-4000-8000-000000000001",
    "name": "Test User",
    "email": "test@example.com",
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Error example (401)**

```json
{
  "error": "unauthorized"
}
```

---

## Projects

All project routes require:

```http
Authorization: Bearer <token>
```

### `GET /projects`

List projects the current user can access (owns the project, or has at least one task in it as **assignee** or **creator** via `created_by`).

| | |
|--|--|
| **Description** | Returns `{ "projects": [...] }`, or adds `total`, `page`, and `limit` when `page` and/or `limit` are present. |
| **Request headers** | `Authorization` |
| **Query parameters** | Optional: `page` (≥ 1), `limit` (≥ 1, capped at 100 server-side when paginating). |

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Success. |
| 400 | Invalid `page` / `limit`. |
| 401 | Missing or invalid token. |
| 500 | Internal error. |

**Response body (200) — without pagination**

```json
{
  "projects": [
    {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Demo Project",
      "description": "Seed project owned by test@example.com",
      "owner_id": "10000000-0000-4000-8000-000000000001",
      "created_at": "2026-04-12T10:00:00Z"
    }
  ]
}
```

**Response body (200) — with pagination** (`?page=1&limit=20`)

```json
{
  "projects": [
    {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Demo Project",
      "description": "Seed project owned by test@example.com",
      "owner_id": "10000000-0000-4000-8000-000000000001",
      "created_at": "2026-04-12T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

**Error example (401)**

```json
{
  "error": "unauthorized"
}
```

---

### `POST /projects`

Create a project; the authenticated user becomes the owner.

| | |
|--|--|
| **Request headers** | `Authorization`, `Content-Type: application/json` |

**Request body (JSON)**

```json
{
  "name": "My project",
  "description": "Optional longer text"
}
```

| Field | Type | Required | Rules |
|-------|------|----------|--------|
| `name` | string | Yes | 1–255 characters |
| `description` | string \| null | No | Max 10000 characters |

**Response codes**

| Code | Description |
|------|-------------|
| 201 | Project created. |
| 400 | Validation error. |
| 401 | Unauthenticated. |
| 500 | Internal error. |

**Response body (201)**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "My project",
  "description": "Optional longer text",
  "owner_id": "10000000-0000-4000-8000-000000000001",
  "created_at": "2026-04-12T12:00:00Z"
}
```

---

### `GET /projects/:id`

Return one project with **all** tasks in that project and `task_count`.

| | |
|--|--|
| **Path** | `id` — project UUID. |

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Success. |
| 400 | Invalid UUID. |
| 401 | Unauthenticated. |
| 403 | No access to project. |
| 404 | Project not found. |
| 500 | Internal error. |

**Response body (200)**

```json
{
  "id": "20000000-0000-4000-8000-000000000001",
  "name": "Demo Project",
  "description": "Seed project owned by test@example.com",
  "owner_id": "10000000-0000-4000-8000-000000000001",
  "created_at": "2026-04-12T10:00:00Z",
  "task_count": 3,
  "tasks": [
    {
      "id": "30000000-0000-4000-8000-000000000001",
      "title": "Backlog item",
      "description": "Task in todo state",
      "status": "todo",
      "priority": "low",
      "project_id": "20000000-0000-4000-8000-000000000001",
      "assignee_id": "10000000-0000-4000-8000-000000000001",
      "created_by": null,
      "due_date": null,
      "created_at": "2026-04-12T10:00:00Z",
      "updated_at": "2026-04-12T10:00:00Z"
    }
  ]
}
```

**Error example (404)**

```json
{
  "error": "not found"
}
```

---

### `GET /projects/:id/stats`

Aggregated task counts by `status` and by assignee user id. Unassigned tasks are counted under the key `__unassigned__`.

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Success. |
| 400 | Invalid UUID. |
| 401 | Unauthenticated. |
| 403 / 404 | No access or not found (per service rules). |
| 500 | Internal error. |

**Response body (200)**

```json
{
  "by_status": {
    "todo": 1,
    "in_progress": 1,
    "done": 1
  },
  "by_assignee": {
    "10000000-0000-4000-8000-000000000001": 2,
    "__unassigned__": 1
  }
}
```

---

### `PATCH /projects/:id`

Partial update. **Owner only.**

**Request body (JSON)** — at least one field; omit fields you do not want to change.

```json
{
  "name": "Renamed project",
  "description": "Updated description"
}
```

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Updated. |
| 400 | Validation / invalid UUID. |
| 401 | Unauthenticated. |
| 403 | Not owner. |
| 404 | Not found. |
| 500 | Internal error. |

**Response body (200)**

```json
{
  "id": "20000000-0000-4000-8000-000000000001",
  "name": "Renamed project",
  "description": "Updated description",
  "owner_id": "10000000-0000-4000-8000-000000000001",
  "created_at": "2026-04-12T10:00:00Z"
}
```

**Error example (403)**

```json
{
  "error": "forbidden"
}
```

---

### `DELETE /projects/:id`

Delete project and its tasks (cascade). **Owner only.**

**Response codes**

| Code | Description |
|------|-------------|
| 204 | Deleted; **empty body**. |
| 400 | Invalid UUID. |
| 401 | Unauthenticated. |
| 403 | Not owner. |
| 404 | Not found. |
| 500 | Internal error. |

---

## Tasks

All task routes require `Authorization: Bearer <token>`.

### `GET /projects/:id/tasks`

List tasks for a project.

| **Query parameters** | Optional |
|---------------------|----------|
| `status` | `todo`, `in_progress`, or `done` |
| `assignee` | User UUID |
| `page`, `limit` | Positive integers; if either is present, response adds `total`, `page`, and `limit` alongside `tasks`. |

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Success. |
| 400 | Invalid query (`status`, `assignee`, `page`, `limit`) or invalid path UUID. |
| 401 | Unauthenticated. |
| 403 / 404 | No access or project not found. |
| 500 | Internal error. |

**Response body (200) — without pagination**

```json
{
  "tasks": [
    {
      "id": "30000000-0000-4000-8000-000000000001",
      "title": "Backlog item",
      "description": "Task in todo state",
      "status": "todo",
      "priority": "low",
      "project_id": "20000000-0000-4000-8000-000000000001",
      "assignee_id": "10000000-0000-4000-8000-000000000001",
      "created_by": null,
      "due_date": null,
      "created_at": "2026-04-12T10:00:00Z",
      "updated_at": "2026-04-12T10:00:00Z"
    }
  ]
}
```

**Response body (200) — with pagination** (`?page=1&limit=20`)

```json
{
  "tasks": [
    {
      "id": "30000000-0000-4000-8000-000000000001",
      "title": "Backlog item",
      "status": "todo",
      "priority": "low",
      "project_id": "20000000-0000-4000-8000-000000000001",
      "assignee_id": "10000000-0000-4000-8000-000000000001",
      "created_by": null,
      "due_date": null,
      "created_at": "2026-04-12T10:00:00Z",
      "updated_at": "2026-04-12T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 20
}
```

**Error example (400) — invalid status**

```json
{
  "error": "validation failed",
  "fields": {
    "status": "invalid task status: \"invalid\""
  }
}
```

---

### `POST /projects/:id/tasks`

Create a task. The path `:id` is the project id; the server **overrides** any `project_id` in the body with the path id (you may omit `project_id` in JSON).

**Request body (JSON)**

```json
{
  "title": "Implement API docs",
  "description": "Optional",
  "status": "todo",
  "priority": "high",
  "assignee_id": "10000000-0000-4000-8000-000000000001",
  "due_date": "2026-04-15T00:00:00Z"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|--------|
| `title` | string | Yes | 1–255 characters |
| `description` | string \| null | No | Max 10000 |
| `status` | string | No | Defaults to `todo` |
| `priority` | string | No | Defaults to `medium` |
| `assignee_id` | UUID \| null | No | Must exist if set |
| `due_date` | string (RFC 3339) \| null | No | |

**Response codes**

| Code | Description |
|------|-------------|
| 201 | Task created. |
| 400 | Validation error. |
| 401 | Unauthenticated. |
| 403 / 404 | No access to project. |
| 500 | Internal error. |

**Response body (201)**

```json
{
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "title": "Implement API docs",
  "description": "Optional",
  "status": "todo",
  "priority": "high",
  "project_id": "20000000-0000-4000-8000-000000000001",
  "assignee_id": "10000000-0000-4000-8000-000000000001",
  "created_by": "10000000-0000-4000-8000-000000000001",
  "due_date": "2026-04-15T00:00:00Z",
  "created_at": "2026-04-12T12:30:00Z",
  "updated_at": "2026-04-12T12:30:00Z"
}
```

---

### `PATCH /tasks/:id`

Partial update. Allowed when you have access to the task’s project (owner or participant per service rules). **At least one** field required.

**Request body (JSON)**

```json
{
  "status": "done",
  "priority": "low"
}
```

**Response codes**

| Code | Description |
|------|-------------|
| 200 | Updated. |
| 400 | Validation / empty body. |
| 401 | Unauthenticated. |
| 403 / 404 | No access or task not found. |
| 500 | Internal error. |

**Response body (200)** — full task object (same shape as create).

**Error example (400) — no fields**

```json
{
  "error": "validation failed",
  "fields": {
    "body": "at least one field must be provided"
  }
}
```

---

### `DELETE /tasks/:id`

Delete a task if you are allowed (project owner or task creator per service rules).

**Response codes**

| Code | Description |
|------|-------------|
| 204 | Deleted; **empty body**. |
| 400 | Invalid UUID. |
| 401 | Unauthenticated. |
| 403 / 404 | Not allowed or not found. |
| 500 | Internal error. |

---

## Enumerations

### Task `status`

| Value | Description |
|-------|-------------|
| `todo` | Not started |
| `in_progress` | In progress |
| `done` | Completed |

### Task `priority`

| Value |
|-------|
| `low` |
| `medium` |
| `high` |

---

## Related files

- Local setup: [SETUP.md](./SETUP.md)
- Postman: [TaskFlow.postman_collection.json](./postman/TaskFlow.postman_collection.json)
- cURL examples: [examples/requests.sh](../examples/requests.sh)
