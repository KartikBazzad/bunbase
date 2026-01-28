#!/bin/bash

# Test script to deploy and test a QuickJS function

set -e

FUNCTION_NAME="quickjs-hello"
FUNCTION_FILE="examples/quickjs-hello.ts"

echo "=== Testing QuickJS-NG Function Deployment ==="
echo ""

# Deploy the function
echo "1. Deploying function..."
./scripts/deploy-quickjs-function.sh "$FUNCTION_NAME" "$FUNCTION_FILE" v1 strict

echo ""
echo "2. Verifying deployment..."
# Check if function was registered
if sqlite3 data/metadata.db "SELECT name FROM functions WHERE name = '$FUNCTION_NAME';" 2>/dev/null | grep -q "$FUNCTION_NAME"; then
    echo "✅ Function registered in database"
else
    echo "❌ Function not found in database"
    exit 1
fi

# Check if bundle exists
BUNDLE_PATH="data/bundles/func-$FUNCTION_NAME/v1/bundle.js"
if [ -f "$BUNDLE_PATH" ]; then
    echo "✅ Bundle file exists: $BUNDLE_PATH"
    echo "   Bundle size: $(du -h "$BUNDLE_PATH" | cut -f1)"
else
    echo "❌ Bundle file not found: $BUNDLE_PATH"
    exit 1
fi

echo ""
echo "=== Deployment Test Complete ==="
echo ""
echo "Next steps:"
echo "  1. Start the functions service:"
echo "     ./functions --data-dir ./data"
echo ""
echo "  2. Test the function:"
echo "     curl 'http://localhost:8080/functions/$FUNCTION_NAME?name=QuickJS'"
echo ""
