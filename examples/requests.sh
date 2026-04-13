#!/usr/bin/env bash
# =============================================================================
# TaskFlow API — cURL examples
# =============================================================================
# Prerequisites: bash (Git Bash / WSL / macOS / Linux), curl, jq (optional).
#
# Usage:
#   1. Start the stack: docker compose up --build
#   2. Export BASE_URL (default below) and obtain a JWT:
#        export TOKEN="$(curl -s -X POST "$BASE_URL/auth/login" ... | jq -r '.data.token')"
#   3. Run individual commands or source this file and copy-paste sections.
#
# Test user (after seed migration): test@example.com / password123
# Seed project id: 20000000-0000-4000-8000-000000000001
# Seed task id:   30000000-0000-4000-8000-000000000001
# =============================================================================

BASE_URL="${BASE_URL:-http://localhost:8080}"

# -----------------------------------------------------------------------------
# Health (no auth)
# -----------------------------------------------------------------------------
# Liveness check — expect {"status":"ok"}

curl -sS "${BASE_URL}/health"
echo

# -----------------------------------------------------------------------------
# Auth — Register (no auth)
# -----------------------------------------------------------------------------
# Success: 201 + data.token + data.user
# Error: 409 if email already exists

curl -sS -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Demo User",
    "email": "demo@example.com",
    "password": "password123"
  }'
echo

# Validation error example (short password) — expect 400

curl -sS -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "X",
    "email": "not-an-email",
    "password": "short"
  }'
echo

# -----------------------------------------------------------------------------
# Auth — Login (no auth) — use this to set TOKEN
# -----------------------------------------------------------------------------
# Success: 200; copy .data.token into TOKEN

curl -sS -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
echo

# With jq (if installed):
# TOKEN="$(curl -sS -X POST "${BASE_URL}/auth/login" \
#   -H "Content-Type: application/json" \
#   -d '{"email":"test@example.com","password":"password123"}' | jq -r '.data.token')"

# Invalid credentials — expect 401 {"error":"invalid credentials"}

curl -sS -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "wrong-password"
  }'
echo

# -----------------------------------------------------------------------------
# Set TOKEN before protected calls (replace with value from login response)
# -----------------------------------------------------------------------------

TOKEN="${TOKEN:-PASTE_JWT_HERE}"

AUTH_HEADER=(-H "Authorization: Bearer ${TOKEN}")

# -----------------------------------------------------------------------------
# Projects
# -----------------------------------------------------------------------------

# List all accessible projects (no pagination) — 200, data = array

curl -sS "${BASE_URL}/projects" "${AUTH_HEADER[@]}"
echo

# List with pagination — 200, data = { items, total, page, limit }

curl -sS "${BASE_URL}/projects?page=1&limit=10" "${AUTH_HEADER[@]}"
echo

# Create project — 201

curl -sS -X POST "${BASE_URL}/projects" \
  "${AUTH_HEADER[@]}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My project",
    "description": "Created via curl"
  }'
echo

# Get project with tasks (seed project)

curl -sS "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001" "${AUTH_HEADER[@]}"
echo

# Project stats

curl -sS "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001/stats" "${AUTH_HEADER[@]}"
echo

# Update project (owner only)

curl -sS -X PATCH "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001" \
  "${AUTH_HEADER[@]}" \
  -H "Content-Type: application/json" \
  -d '{"name": "Demo Project (updated)"}'
echo

# Invalid path UUID — 400

curl -sS "${BASE_URL}/projects/not-a-uuid" "${AUTH_HEADER[@]}"
echo

# No Authorization header — 401

curl -sS "${BASE_URL}/projects"
echo

# -----------------------------------------------------------------------------
# Tasks
# -----------------------------------------------------------------------------

# List tasks for seed project

curl -sS "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001/tasks" "${AUTH_HEADER[@]}"
echo

# Filter by status + pagination

curl -sS "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001/tasks?status=todo&page=1&limit=5" "${AUTH_HEADER[@]}"
echo

# Invalid status filter — 400

curl -sS "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001/tasks?status=invalid" "${AUTH_HEADER[@]}"
echo

# Create task (project_id in body is optional; path sets project)

curl -sS -X POST "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001/tasks" \
  "${AUTH_HEADER[@]}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "New task from curl",
    "description": "Optional",
    "status": "todo",
    "priority": "medium"
  }'
echo

# Update task (use a real task id from list/create; example uses seed todo task)

curl -sS -X PATCH "${BASE_URL}/tasks/30000000-0000-4000-8000-000000000001" \
  "${AUTH_HEADER[@]}" \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress", "priority": "high"}'
echo

# Delete task — 204 empty body (replace TASK_ID)

# curl -sS -o /dev/null -w "%{http_code}\n" -X DELETE "${BASE_URL}/tasks/TASK_ID" "${AUTH_HEADER[@]}"

# Delete project — 204 empty body (destructive; owner only)

# curl -sS -o /dev/null -w "%{http_code}\n" -X DELETE "${BASE_URL}/projects/20000000-0000-4000-8000-000000000001" "${AUTH_HEADER[@]}"
