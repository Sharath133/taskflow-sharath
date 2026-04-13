#!/bin/sh
# =============================================================================
# Wait for PostgreSQL to accept connections, then exec golang-migrate.
# =============================================================================
# Compose already waits for `postgres` to be healthy; this loop handles the rare
# race where migrate still cannot dial. `migrate version` exits 1 with
# "no migration" on a fresh database even though the server is reachable.

set -eu

MIGRATE_BIN="migrate"
MIGRATE_PATH="/migrations"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <migrate subcommand> [args...]" >&2
  exit 1
fi

echo "Waiting for PostgreSQL at migrate CLI (${MIGRATE_PATH})..."
i=0
max=60
while [ "$i" -lt "$max" ]; do
  if version_out="$("${MIGRATE_BIN}" -path="${MIGRATE_PATH}" -database="${DATABASE_URL}" version 2>&1)"; then
    echo "PostgreSQL is reachable; running: ${MIGRATE_BIN} $*"
    exec "${MIGRATE_BIN}" -path="${MIGRATE_PATH}" -database="${DATABASE_URL}" "$@"
  fi
  case ${version_out} in *"no migration"*)
    echo "PostgreSQL is reachable (fresh database); running: ${MIGRATE_BIN} $*"
    exec "${MIGRATE_BIN}" -path="${MIGRATE_PATH}" -database="${DATABASE_URL}" "$@"
    ;;
  esac
  i=$((i + 1))
  sleep 1
done

echo "Timed out waiting for PostgreSQL (${max}s)." >&2
exit 1
