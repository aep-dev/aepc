#!/usr/bin/env bash
set -e

echo "Testing Publisher Undelete Functionality"
echo "========================================"

# Wait a moment for server to be ready
sleep 2

PUBLISHER_ID="test-undelete-publisher"
BASE_URL="http://localhost:8081"

echo "1. Creating a publisher..."
curl -X POST "${BASE_URL}/publishers?id=${PUBLISHER_ID}" \
    -H "Content-Type: application/json" \
    -d '{"description": "Test Publisher for Undelete Demo"}' \
    -s -o /dev/null -w "HTTP Status: %{http_code}\n"

echo "2. Verifying publisher was created..."
PUBLISHER=$(curl -s "${BASE_URL}/publishers/${PUBLISHER_ID}")
echo "Publisher: $PUBLISHER"

echo "3. Deleting the publisher (moves to deleted_publishers)..."
curl -X DELETE "${BASE_URL}/publishers/${PUBLISHER_ID}" \
    -s -o /dev/null -w "HTTP Status: %{http_code}\n"

echo "4. Verifying publisher is no longer accessible..."
curl -s "${BASE_URL}/publishers/${PUBLISHER_ID}" -w "HTTP Status: %{http_code}\n" || echo "Expected 404 - publisher not found"

echo "5. Checking deleted publishers..."
DELETED_PUBLISHERS=$(curl -s "${BASE_URL}/deleted_publishers")
echo "Deleted Publishers: $DELETED_PUBLISHERS"

echo "6. Getting the specific deleted publisher..."
DELETED_PUBLISHER=$(curl -s "${BASE_URL}/deleted_publishers/${PUBLISHER_ID}")
echo "Deleted Publisher: $DELETED_PUBLISHER"

echo "7. Undeleting the publisher..."
UNDELETE_RESPONSE=$(curl -X POST "${BASE_URL}/deleted_publishers/${PUBLISHER_ID}:undelete" \
    -H "Content-Type: application/json" \
    -d '{}' \
    -s -w "HTTP Status: %{http_code}")
echo "Undelete response: $UNDELETE_RESPONSE"

echo "8. Verifying publisher is restored..."
RESTORED_PUBLISHER=$(curl -s "${BASE_URL}/publishers/${PUBLISHER_ID}")
echo "Restored Publisher: $RESTORED_PUBLISHER"

echo "9. Verifying publisher is no longer in deleted_publishers..."
curl -s "${BASE_URL}/deleted_publishers/${PUBLISHER_ID}" -w "HTTP Status: %{http_code}\n" || echo "Expected 404 - deleted publisher not found"

echo ""
echo "Undelete functionality test completed successfully!"