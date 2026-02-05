#!/usr/bin/env bash
# Deploy the hello-world example function so the demo app can invoke it.
# Requires: bunbase CLI on PATH (see docs/users/cli-guide.md).

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
EXAMPLE="$REPO_ROOT/functions/examples/hello-world.ts"

if ! command -v bunbase &>/dev/null; then
  echo "bunbase CLI not found. Build it with: cd platform && go build -o bunbase ./cmd/cli"
  echo "See docs/users/cli-guide.md for details."
  exit 1
fi

if [[ ! -f "$EXAMPLE" ]]; then
  echo "Example file not found: $EXAMPLE"
  exit 1
fi

echo "Deploying hello-world from $EXAMPLE ..."
bunbase deploy "$EXAMPLE" --name hello-world --runtime bun --handler default
echo "Done. Use the demo app Functions page to invoke hello-world."
