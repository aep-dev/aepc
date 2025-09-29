#!/bin/bash

# Integration test for If-Match header functionality
# This script demonstrates the If-Match header working with the HTTP API

set -e

echo "Starting integration test for If-Match header functionality..."

# Kill any existing processes on port 8081
lsof -ti:8081 | xargs kill -9 2>/dev/null || true

# Start the server in the background
go run example/main.go &
SERVER_PID=$!

# Wait for server to start and check if it's running
echo "Waiting for server to start..."
for i in {1..10}; do
    if curl -s http://localhost:8081/openapi.json > /dev/null 2>&1; then
        echo "Server started successfully"
        break
    fi
    if [ "$i" -eq 10 ]; then
        echo "Server failed to start after 10 seconds"
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
}
trap cleanup EXIT

echo "Testing If-Match header functionality..."

# 1. Create a publisher and capture the ETag from the response headers
echo "Creating publisher..."
PUBLISHER_RESPONSE=$(curl -s -i -X POST "http://localhost:8081/publishers" \
  -H "Content-Type: application/json" \
  -d '{"description": "Test Publisher for If-Match"}')

echo "Publisher creation response:"
echo "$PUBLISHER_RESPONSE"

# Extract the JSON body (after the empty line that separates headers from body)
PUBLISHER_JSON=$(echo "$PUBLISHER_RESPONSE" | sed -n '/^\r*$/,$p' | tail -n +2 | tr -d '\r')
echo "Publisher JSON: '$PUBLISHER_JSON'"

# Extract the path from the response body
PUBLISHER_PATH=$(echo "$PUBLISHER_JSON" | jq -r '.path')
echo "Created publisher: '$PUBLISHER_PATH'"

# Extract ETag from headers (case insensitive)
ETAG=$(echo "$PUBLISHER_RESPONSE" | grep -i "etag:" | cut -d' ' -f2- | tr -d '\r' | xargs)
echo "ETag from creation: '$ETAG'"

if [ -z "$ETAG" ]; then
    echo "ERROR: No ETag header found in create response"
    exit 1
fi

# 2. Get the publisher to retrieve current ETag
echo "Getting current publisher to verify ETag..."
GET_RESPONSE=$(curl -s -i "http://localhost:8081/${PUBLISHER_PATH}")
echo "Get response:"
echo "$GET_RESPONSE"

# Extract ETag from GET response
GET_ETAG=$(echo "$GET_RESPONSE" | grep -i "etag:" | cut -d' ' -f2- | tr -d '\r' | xargs)
echo "ETag from GET: '$GET_ETAG'"

if [ -z "$GET_ETAG" ]; then
    echo "ERROR: No ETag header found in GET response"
    exit 1
fi

echo "Testing update with incorrect ETag (should fail)..."
WRONG_ETAG='"wrong-etag-value"'
WRONG_UPDATE_RESPONSE=$(curl -s -i -X PATCH "http://localhost:8081/${PUBLISHER_PATH}" \
  -H "Content-Type: application/json" \
  -H "If-Match: $WRONG_ETAG" \
  -d '{"description": "Updated description - should fail"}')

echo "Response with wrong ETag:"
echo "$WRONG_UPDATE_RESPONSE"

# Check if the response contains an error status
if echo "$WRONG_UPDATE_RESPONSE" | head -n 1 | grep -q "400"; then
    echo "✅ PASS: Update with wrong ETag was correctly rejected"
else
    echo "❌ FAIL: Update with wrong ETag should have been rejected but wasn't"
    exit 1
fi

# 4. Update with the correct ETag (should succeed)
echo "Testing update with correct ETag (should succeed)..."
CORRECT_UPDATE_RESPONSE=$(curl -s -i -X PATCH "http://localhost:8081/${PUBLISHER_PATH}" \
  -H "Content-Type: application/json" \
  -H "If-Match: $GET_ETAG" \
  -d '{"description": "Updated description - should succeed"}')

echo "Response with correct ETag:"
echo "$CORRECT_UPDATE_RESPONSE"

# Check if the response was successful
if echo "$CORRECT_UPDATE_RESPONSE" | head -n 1 | grep -q "200"; then
    echo "✅ PASS: Update with correct ETag was successful"

    # Extract new ETag from update response (take the last one in case of duplicates)
    NEW_ETAG=$(echo "$CORRECT_UPDATE_RESPONSE" | grep -i "etag:" | tail -n 1 | cut -d' ' -f2- | tr -d '\r' | xargs)
    echo "New ETag after update: '$NEW_ETAG'"

    # Verify the description was actually updated
    UPDATED_JSON=$(echo "$CORRECT_UPDATE_RESPONSE" | sed -n '/^\r*$/,$p' | tail -n +2 | tr -d '\r')
    UPDATED_DESCRIPTION=$(echo "$UPDATED_JSON" | jq -r '.description')
    echo "Updated description: '$UPDATED_DESCRIPTION'"

    if [ "$UPDATED_DESCRIPTION" = "Updated description - should succeed" ]; then
        echo "✅ PASS: Publisher description was correctly updated"
    else
        echo "❌ FAIL: Publisher description was not updated correctly"
        exit 1
    fi

    # Verify that the ETag changed after the update
    if [ "$NEW_ETAG" != "$GET_ETAG" ]; then
        echo "✅ PASS: ETag changed after update"
    else
        echo "❌ FAIL: ETag should have changed after update"
        exit 1
    fi
else
    echo "❌ FAIL: Update with correct ETag should have succeeded but didn't"
    exit 1
fi

# 5. Try to update again with the old ETag (should fail)
echo "Testing update with old ETag after the resource changed (should fail)..."
OLD_ETAG_RESPONSE=$(curl -s -i -X PATCH "http://localhost:8081/${PUBLISHER_PATH}" \
  -H "Content-Type: application/json" \
  -H "If-Match: $GET_ETAG" \
  -d '{"description": "This update should fail"}')

echo "Response with old ETag:"
echo "$OLD_ETAG_RESPONSE"

if echo "$OLD_ETAG_RESPONSE" | head -n 1 | grep -q "400"; then
    echo "✅ PASS: Update with old ETag was correctly rejected after resource changed"
else
    echo "❌ FAIL: Update with old ETag should have been rejected after resource changed"
    exit 1
fi

echo ""
echo "🎉 All If-Match header tests passed!"
echo "✅ ETag headers are returned on CREATE, GET, and UPDATE operations"
echo "✅ If-Match header validation works correctly"
echo "✅ Updates with wrong ETags are rejected"
echo "✅ Updates with correct ETags succeed"
echo "✅ ETags change after resource updates"