#!/usr/bin/env bash
# Deploy the todo-processor function from demo-app.
# Requires: bunbase CLI on PATH (see docs/users/cli-guide.md).

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FUNCTION_FILE="$REPO_ROOT/demo-app/src/functions/todo-processor.ts"

if ! command -v bunbase &>/dev/null; then
  echo "bunbase CLI not found. Build it with: cd platform && go build -o bunbase ./cmd/cli"
  echo "See docs/users/cli-guide.md for details."
  exit 1
fi

if [[ ! -f "$FUNCTION_FILE" ]]; then
  echo "Function file not found: $FUNCTION_FILE"
  exit 1
fi

echo "Deploying todo-processor from $FUNCTION_FILE ..."
bunbase deploy "$FUNCTION_FILE" --name todo-processor --runtime bun --handler default
echo "Done. Use the demo app Functions page to invoke todo-processor."
