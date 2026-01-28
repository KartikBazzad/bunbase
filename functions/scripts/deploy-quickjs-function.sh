#!/bin/bash

# Deploy a function with QuickJS-NG runtime
# Usage: ./scripts/deploy-quickjs-function.sh <function-name> <function-file> [version] [capability-profile]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
FUNCTION_NAME="${1:-}"
FUNCTION_FILE="${2:-}"
VERSION="${3:-v1}"
CAPABILITY_PROFILE="${4:-strict}"  # strict, permissive, or custom
DATA_DIR="${DATA_DIR:-./data}"
DB_PATH="$DATA_DIR/metadata.db"
SOCKET_PATH="${SOCKET_PATH:-/tmp/functions.sock}"

# Validate inputs
if [ -z "$FUNCTION_NAME" ] || [ -z "$FUNCTION_FILE" ]; then
    echo -e "${RED}Usage: $0 <function-name> <function-file> [version] [capability-profile]${NC}"
    echo ""
    echo "Examples:"
    echo "  $0 hello-world examples/hello-world.ts"
    echo "  $0 my-function ./my-function.ts v2 strict"
    echo "  $0 api-handler ./api.ts v1 permissive"
    echo ""
    exit 1
fi

if [ ! -f "$FUNCTION_FILE" ]; then
    echo -e "${RED}❌ Function file not found: $FUNCTION_FILE${NC}"
    exit 1
fi

FUNCTION_ID="func-$(echo "$FUNCTION_NAME" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')"
BUNDLE_DIR="$DATA_DIR/bundles/$FUNCTION_ID/$VERSION"
BUNDLE_PATH="$BUNDLE_DIR/bundle.js"

echo -e "${GREEN}=== Deploying Function with QuickJS-NG Runtime ===${NC}"
echo ""
echo "Function Name: $FUNCTION_NAME"
echo "Function ID:   $FUNCTION_ID"
echo "Version:       $VERSION"
echo "Runtime:       quickjs-ng"
echo "Capabilities:  $CAPABILITY_PROFILE"
echo "Source File:   $FUNCTION_FILE"
echo ""

# 1. Create bundle directory
echo "1. Creating bundle directory..."
mkdir -p "$BUNDLE_DIR"

# 2. Build function bundle for QuickJS
echo "2. Building function bundle for QuickJS-NG..."
echo "   Note: QuickJS-NG supports ES modules and modern JavaScript"

BUNDLE_BUILT=false

# Check if bun is available for bundling
if command -v bun &> /dev/null; then
    echo "   Using Bun to bundle (targeting QuickJS-compatible output)..."
    # Build with browser target for QuickJS compatibility (no Node.js APIs)
    if bun build "$FUNCTION_FILE" \
        --outdir "$BUNDLE_DIR" \
        --target browser \
        --minify \
        --outfile bundle.js 2>/dev/null; then
        BUNDLE_BUILT=true
        # Check if bundle was created with different name
        if [ -f "$BUNDLE_DIR/$(basename "$FUNCTION_FILE" .ts).js" ]; then
            mv "$BUNDLE_DIR/$(basename "$FUNCTION_FILE" .ts).js" "$BUNDLE_PATH"
        fi
    else
        echo -e "${YELLOW}   ⚠️  Bun build failed, trying esbuild...${NC}"
    fi
fi

# Fallback to esbuild if bun failed or not available
if [ "$BUNDLE_BUILT" = false ] && command -v esbuild &> /dev/null; then
    echo "   Using esbuild to bundle..."
    if esbuild "$FUNCTION_FILE" \
        --bundle \
        --platform=browser \
        --format=esm \
        --outfile="$BUNDLE_PATH" \
        --minify 2>/dev/null; then
        BUNDLE_BUILT=true
    else
        echo -e "${YELLOW}   ⚠️  esbuild build failed${NC}"
    fi
fi

# Last resort: if it's already a .js file, just copy it
if [ "$BUNDLE_BUILT" = false ]; then
    if [[ "$FUNCTION_FILE" == *.js ]]; then
        echo "   Copying JavaScript file as bundle..."
        cp "$FUNCTION_FILE" "$BUNDLE_PATH"
        BUNDLE_BUILT=true
    else
        echo -e "${RED}❌ Failed to build bundle. Please ensure bun or esbuild is installed${NC}"
        echo "   You can also provide a pre-built .js file"
        exit 1
    fi
fi

if [ ! -f "$BUNDLE_PATH" ]; then
    echo -e "${RED}❌ Bundle not created at $BUNDLE_PATH${NC}"
    exit 1
fi

echo -e "${GREEN}   ✅ Bundle created at $BUNDLE_PATH${NC}"

# 3. Get absolute path
BUNDLE_ABS_PATH=$(cd "$BUNDLE_DIR" && pwd)/bundle.js
echo "   Bundle absolute path: $BUNDLE_ABS_PATH"

# 4. Determine capabilities JSON based on profile
case "$CAPABILITY_PROFILE" in
    strict)
        CAPABILITIES_JSON='{"AllowFilesystem":false,"AllowNetwork":false,"AllowChildProcess":false,"AllowEval":false,"AllowedPaths":null,"AllowedDomains":null,"MaxMemory":104857600,"MaxCPU":30000000000,"MaxFileDescriptors":10,"ProjectID":"'$FUNCTION_ID'"}'
        ;;
    permissive)
        CAPABILITIES_JSON='{"AllowFilesystem":true,"AllowNetwork":true,"AllowChildProcess":true,"AllowEval":true,"AllowedPaths":null,"AllowedDomains":null,"MaxMemory":536870912,"MaxCPU":300000000000,"MaxFileDescriptors":100,"ProjectID":"'$FUNCTION_ID'"}'
        ;;
    *)
        echo -e "${YELLOW}⚠️  Unknown capability profile '$CAPABILITY_PROFILE', using strict${NC}"
        CAPABILITIES_JSON='{"AllowFilesystem":false,"AllowNetwork":false,"AllowChildProcess":false,"AllowEval":false,"AllowedPaths":null,"AllowedDomains":null,"MaxMemory":104857600,"MaxCPU":30000000000,"MaxFileDescriptors":10,"ProjectID":"'$FUNCTION_ID'"}'
        ;;
esac

# 5. Ensure database exists and has schema
echo "3. Ensuring database schema..."
if [ ! -f "$DB_PATH" ]; then
    mkdir -p "$(dirname "$DB_PATH")"
    echo "   Creating new database..."
fi

# 6. Migrate schema if needed (add capabilities_json column if missing)
echo "4. Checking database schema..."
HAS_CAPABILITIES_COLUMN=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM pragma_table_info('functions') WHERE name='capabilities_json';" 2>/dev/null || echo "0")

if [ "$HAS_CAPABILITIES_COLUMN" = "0" ]; then
    echo "   Migrating schema: adding capabilities_json column..."
    sqlite3 "$DB_PATH" "ALTER TABLE functions ADD COLUMN capabilities_json TEXT;" 2>/dev/null || {
        # If table doesn't exist, create it with full schema
        echo "   Creating tables with full schema..."
        sqlite3 "$DB_PATH" <<EOF
CREATE TABLE IF NOT EXISTS functions (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    runtime TEXT NOT NULL,
    handler TEXT NOT NULL,
    status TEXT NOT NULL,
    active_version_id TEXT,
    capabilities_json TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS function_versions (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    version TEXT NOT NULL,
    bundle_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (function_id) REFERENCES functions(id),
    UNIQUE(function_id, version)
);

CREATE TABLE IF NOT EXISTS function_deployments (
    id TEXT PRIMARY KEY,
    function_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (function_id) REFERENCES functions(id),
    FOREIGN KEY (version_id) REFERENCES function_versions(id)
);

CREATE INDEX IF NOT EXISTS idx_functions_name ON functions(name);
CREATE INDEX IF NOT EXISTS idx_versions_function_id ON function_versions(function_id);
CREATE INDEX IF NOT EXISTS idx_deployments_function_id ON function_deployments(function_id);
EOF
    }
    echo -e "${GREEN}   ✅ Schema migration complete${NC}"
else
    echo "   Schema is up to date"
fi

# 7. Insert into database
echo "5. Registering function in database..."

sqlite3 "$DB_PATH" <<EOF
-- Insert or replace function with QuickJS-NG runtime
INSERT OR REPLACE INTO functions (id, name, runtime, handler, status, capabilities_json, created_at, updated_at)
VALUES ('$FUNCTION_ID', '$FUNCTION_NAME', 'quickjs-ng', 'handler', 'registered', '$CAPABILITIES_JSON', strftime('%s', 'now'), strftime('%s', 'now'));

-- Insert or replace version
INSERT OR REPLACE INTO function_versions (id, function_id, version, bundle_path, created_at)
VALUES ('version-$VERSION', '$FUNCTION_ID', '$VERSION', '$BUNDLE_ABS_PATH', strftime('%s', 'now'));

-- Insert or replace deployment
INSERT OR REPLACE INTO function_deployments (id, function_id, version_id, status, created_at)
VALUES ('deploy-$VERSION', '$FUNCTION_ID', 'version-$VERSION', 'active', strftime('%s', 'now'));

-- Update function to deployed
UPDATE functions 
SET status = 'deployed', active_version_id = 'version-$VERSION', updated_at = strftime('%s', 'now')
WHERE id = '$FUNCTION_ID';
EOF

echo -e "${GREEN}   ✅ Function registered and deployed${NC}"

# 8. Verify deployment
echo "6. Verifying deployment..."
FUNCTION_STATUS=$(sqlite3 "$DB_PATH" "SELECT status FROM functions WHERE id = '$FUNCTION_ID';" 2>/dev/null || echo "")
if [ "$FUNCTION_STATUS" = "deployed" ]; then
    echo -e "${GREEN}   ✅ Function is deployed${NC}"
else
    echo -e "${YELLOW}   ⚠️  Function status: $FUNCTION_STATUS${NC}"
fi

echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo ""
echo "Function Details:"
echo "  ID:       $FUNCTION_ID"
echo "  Name:     $FUNCTION_NAME"
echo "  Version:  $VERSION"
echo "  Runtime:  quickjs-ng"
echo "  Bundle:   $BUNDLE_ABS_PATH"
echo "  Status:   deployed"
echo ""
echo "Next Steps:"
echo "  1. Start the functions service (if not running):"
echo "     ./functions --data-dir $DATA_DIR --socket $SOCKET_PATH"
echo ""
echo "  2. The function will be automatically loaded on service start"
echo "     or you can restart the service to load it immediately"
echo ""
echo "  3. Test the function:"
echo "     curl 'http://localhost:8080/functions/$FUNCTION_NAME?name=Alice'"
echo ""
echo "  4. Or use the IPC client:"
echo "     # See pkg/client for Go client usage"
echo ""
