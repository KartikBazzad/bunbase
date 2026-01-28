#!/bin/bash

# Test the worker script directly to see if it works

BUNDLE_PATH="${1:-data/bundles/test-func/v1/bundle.js}"
WORKER_ID="test-worker-123"

if [ ! -f "$BUNDLE_PATH" ]; then
    echo "❌ Bundle not found: $BUNDLE_PATH"
    exit 1
fi

if [ ! -f "worker/worker.ts" ]; then
    echo "❌ Worker script not found: worker/worker.ts"
    exit 1
fi

echo "Testing worker script directly..."
echo "Bundle: $BUNDLE_PATH"
echo ""

# Set environment and run worker
export BUNDLE_PATH="$BUNDLE_PATH"
export WORKER_ID="$WORKER_ID"

# Test if Bun can run the worker
echo "Running: bun worker/worker.ts"
echo ""

# Send a test invoke message after READY
(
    sleep 1  # Wait for READY
    echo '{"id":"test-1","type":"invoke","payload":{"method":"GET","path":"/","headers":{},"query":{"name":"Test"},"body":"","deadline_ms":5000}}'
) | BUNDLE_PATH="$BUNDLE_PATH" WORKER_ID="$WORKER_ID" bun worker/worker.ts
