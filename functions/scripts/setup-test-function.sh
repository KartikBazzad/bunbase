#!/bin/bash

# Setup script to create a test function for manual testing

set -e

FUNCTION_ID="test-func"
VERSION="v1"
BUNDLE_DIR="data/bundles/$FUNCTION_ID/$VERSION"
DB_PATH="data/metadata.db"

echo "=== Setting up test function ==="

# 1. Create bundle directory
echo "1. Creating bundle directory..."
mkdir -p "$BUNDLE_DIR"

# 2. Build example function
echo "2. Building function bundle..."
if [ ! -f "examples/hello-world.ts" ]; then
    echo "❌ examples/hello-world.ts not found"
    exit 1
fi

# Build to the bundle directory (bun creates hello-world.js by default)
bun build examples/hello-world.ts --outdir "$BUNDLE_DIR" --target bun

# Check if bundle was created and rename to bundle.js
if [ -f "$BUNDLE_DIR/hello-world.js" ]; then
    mv "$BUNDLE_DIR/hello-world.js" "$BUNDLE_DIR/bundle.js"
    echo "✅ Bundle created at $BUNDLE_DIR/bundle.js"
elif [ -f "$BUNDLE_DIR/bundle.js" ]; then
    echo "✅ Bundle already exists at $BUNDLE_DIR/bundle.js"
else
    echo "❌ Failed to build bundle"
    echo "Files in $BUNDLE_DIR:"
    ls -la "$BUNDLE_DIR" || echo "Directory doesn't exist"
    exit 1
fi

# 3. Get absolute path
BUNDLE_PATH=$(cd "$BUNDLE_DIR" && pwd)/bundle.js
echo "   Bundle path: $BUNDLE_PATH"

# 4. Insert into database
echo "3. Inserting into database..."

sqlite3 "$DB_PATH" <<EOF
-- Insert function
INSERT OR REPLACE INTO functions (id, name, runtime, handler, status, created_at, updated_at)
VALUES ('$FUNCTION_ID', '$FUNCTION_ID', 'bun', 'handler', 'registered', strftime('%s', 'now'), strftime('%s', 'now'));

-- Insert version
INSERT OR REPLACE INTO function_versions (id, function_id, version, bundle_path, created_at)
VALUES ('version-$VERSION', '$FUNCTION_ID', '$VERSION', '$BUNDLE_PATH', strftime('%s', 'now'));

-- Deploy function
INSERT OR REPLACE INTO function_deployments (id, function_id, version_id, status, created_at)
VALUES ('deploy-$VERSION', '$FUNCTION_ID', 'version-$VERSION', 'active', strftime('%s', 'now'));

-- Update function to deployed
UPDATE functions 
SET status = 'deployed', active_version_id = 'version-$VERSION', updated_at = strftime('%s', 'now')
WHERE id = '$FUNCTION_ID';
EOF

echo "✅ Function registered and deployed"

echo ""
echo "=== Setup complete ==="
echo ""
echo "Note: You still need to create a worker pool for this function."
echo "The pool will be created automatically on first invocation, or you can"
echo "modify main.go to auto-create pools for deployed functions."
echo ""
echo "Test with:"
echo "  curl 'http://localhost:8080/functions/test-func?name=Alice'"
