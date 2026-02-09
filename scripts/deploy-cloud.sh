#!/usr/bin/env bash
# Deploy BunBase in Cloud mode (default: any user can sign up and create projects).
# Run from repo root: ./scripts/deploy-cloud.sh

set -e
cd "$(dirname "$0")/.."

echo "Deploying BunBase (Cloud mode)..."
docker compose up -d

echo ""
echo "Waiting for platform to be healthy..."
for i in {1..30}; do
  if curl -sf http://localhost:3001/health >/dev/null 2>&1; then
    echo "Platform is up."
    break
  fi
  if [[ $i -eq 30 ]]; then
    echo "Platform did not become healthy in time. Check: docker compose logs platform"
    exit 1
  fi
  sleep 2
done

echo ""
echo "Dashboard:  http://localhost"
echo "API:        http://localhost:3001 (or http://localhost/api via Traefik)"
echo ""
echo "Sign up at http://localhost and create a project."
