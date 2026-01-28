#!/bin/bash

# Test function invocation

FUNCTION_NAME="test-func"
NAME="${1:-World}"

echo "Testing function: $FUNCTION_NAME"
echo "Parameter: name=$NAME"
echo ""

# Add timeout to curl
curl -X POST "http://localhost:8080/functions/$FUNCTION_NAME?name=$NAME" \
  -H "Content-Type: application/json" \
  --max-time 60 \
  -v \
  -w "\n\nHTTP Status: %{http_code}\n" 2>&1

echo ""
echo "Test complete!"
