#!/usr/bin/env bash
# Verify platform DB migrations have run (same DB as bun-auth).
# Run from repo root: ./scripts/verify-migrations.sh

set -e
cd "$(dirname "$0")/.."

echo "Checking schema_migrations (platform migration version)..."
VERSION=$(docker compose exec postgres psql -U bunadmin -d bunbase_system -t -A -c "SELECT version FROM schema_migrations;" 2>/dev/null | tr -d '[:space:]') || true
if [[ -z "$VERSION" ]]; then
  echo "Could not read schema_migrations (table may not exist yet)."
  echo "Restart platform after Postgres is healthy: docker compose up -d platform"
  exit 1
fi
DIRTY=$(docker compose exec postgres psql -U bunadmin -d bunbase_system -t -A -c "SELECT dirty FROM schema_migrations;" 2>/dev/null | tr -d '[:space:]') || true
echo "  version=$VERSION dirty=$DIRTY"
if [[ "$DIRTY" == "t" ]]; then
  echo "Migration is dirty. Restart platform to retry or fix."
  exit 1
fi
if [[ "$VERSION" -lt 4 ]] 2>/dev/null; then
  echo "Migration version $VERSION < 4. Run migrations (restart platform)."
  exit 1
fi

echo ""
echo "Checking project_members table exists..."
docker compose exec postgres psql -U bunadmin -d bunbase_system -t -c "SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'project_members';" | grep -q 1 || {
  echo "project_members table not found. Run migrations by restarting platform."
  exit 1
}

echo "Migrations look OK."
