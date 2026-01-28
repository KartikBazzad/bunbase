#!/bin/bash

# Simple test script for BunBase Functions

set -e

echo "=== Testing BunBase Functions ==="

# 1. Test health endpoint
echo ""
echo "1. Testing health endpoint..."
HEALTH=$(curl -s http://localhost:8080/health)
if [ "$HEALTH" = "OK" ]; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed: $HEALTH"
    exit 1
fi

# 2. Check socket exists
echo ""
echo "2. Checking Unix socket..."
if [ -S "/tmp/functions.sock" ]; then
    echo "✅ Socket exists"
else
    echo "❌ Socket not found at /tmp/functions.sock"
    exit 1
fi

# 3. Test function invocation (if function exists)
echo ""
echo "3. Testing function invocation..."
RESPONSE=$(curl -s -X POST "http://localhost:8080/functions/test-func?name=Test" || echo "ERROR")
if [[ "$RESPONSE" == *"message"* ]] || [[ "$RESPONSE" == *"Function not found"* ]] || [[ "$RESPONSE" == *"not deployed"* ]]; then
    echo "✅ Function endpoint responding"
    echo "   Response: $RESPONSE"
else
    echo "⚠️  Function endpoint returned: $RESPONSE"
    echo "   (This is OK if no functions are registered yet)"
fi

echo ""
echo "=== Basic tests complete ==="
echo ""
echo "To test with a real function:"
echo "1. Create a function bundle"
echo "2. Register it in the database (see TESTING.md)"
echo "3. Invoke via: curl http://localhost:8080/functions/your-function-name"
