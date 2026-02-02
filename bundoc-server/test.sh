#!/bin/bash

# Test script for bundoc-server

SERVER_URL="http://localhost:8080"

echo "🧪 Testing Bundoc Server..."
echo ""

# Health check
echo "1. Health Check"
curl -s $SERVER_URL/health | jq .
echo ""
echo ""

# Project 1: my-app
echo "2. Create document in project 'my-app'"
curl -s -X POST $SERVER_URL/v1/projects/my-app/databases/\(default\)/documents/users \
  -H "Content-Type: application/json" \
  -d '{"_id":"user1","name":"Alice","email":"alice@example.com","age":30}' | jq .
echo ""
echo ""

echo "3. Get document from project 'my-app'"
curl -s $SERVER_URL/v1/projects/my-app/databases/\(default\)/documents/users/user1 | jq .
echo ""
echo ""

# Project 2: another-app (isolation test)
echo "4. Create document in different project 'another-app'"
curl -s -X POST $SERVER_URL/v1/projects/another-app/databases/\(default\)/documents/users \
  -H "Content-Type: application/json" \
  -d '{"_id":"user1","name":"Bob","email":"bob@example.com","age":25}' | jq .
echo ""
echo ""

echo "5. Verify project isolation - Get from 'my-app' (should still be Alice)"
curl -s $SERVER_URL/v1/projects/my-app/databases/\(default\)/documents/users/user1 | jq .
echo ""
echo ""

echo "6. Verify project isolation - Get from 'another-app' (should be Bob)"
curl -s $SERVER_URL/v1/projects/another-app/databases/\(default\)/documents/users/user1 | jq .
echo ""
echo ""

# Update test
echo "7. Update document in 'my-app'"
curl -s -X PATCH $SERVER_URL/v1/projects/my-app/databases/\(default\)/documents/users/user1 \
  -H "Content-Type: application/json" \
  -d '{"_id":"user1","name":"Alice Updated","age":31}' | jq .
echo ""
echo ""

# Delete test
echo "8. Delete document from 'another-app'"
curl -s -X DELETE $SERVER_URL/v1/projects/another-app/databases/\(default\)/documents/users/user1 \
  -w "\nStatus: %{http_code}\n"
echo ""
echo ""

# Verify deletion
echo "9. Verify deletion (should return 404)"
curl -s $SERVER_URL/v1/projects/another-app/databases/\(default\)/documents/users/user1 \
  -w "\nStatus: %{http_code}\n" | jq .
echo ""
echo ""

# Final health check
echo "10. Final health check (should show 2 instances - note: may evict if idle)"
curl -s $SERVER_URL/health | jq .
echo ""

echo "✅ All tests complete!"
