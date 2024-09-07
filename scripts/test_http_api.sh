#!/usr/bin/env bash
set -ex

GRPC_GATEWAY_PORT=8081

# Get OpenAPI Json
BOOK_ID="tomorrow-and-tomorrow-and-tomorrow"
ISBN="978-1476788036"

# start a process and get it's PID
go run example/main.go &
PID=$!
echo "started server with PID: ${PID}"
sleep 1;

# set a trap, kill the process when the script exits
trap "kill ${PID}" EXIT

# check if "bookstore.example.com" is in OPENAPI_OUTPUT
OPENAPI_OUTPUT=$(curl "http://localhost:${GRPC_GATEWAY_PORT}/openapi.json")
if ! [[ $OPENAPI_OUTPUT == *"bookstore.example.com"* ]]; then
    echo "'bookstore.example.com' not found in OPENAPI_OUTPUT"
fi

curl "http://localhost:8081/books?id=${BOOK_ID}" -X POST -d "{}"
BOOK=$(curl "http://localhost:8081/books/${BOOK_ID}")

# check if "tomorrow-and-tomorrow-and-tomorrow" is in BOOK
if ! [[ $BOOK == *"${BOOK_ID}"* ]]; then
    echo "'${BOOK_ID}' not found in BOOK"
fi

# patch
if [[ $BOOK == *"${BOOK_ID}"* ]]; then
    echo "Patching book"
    curl "http://localhost:8081/books/${BOOK_ID}" -X PATCH -d "{\"isbn\": \"${ISBN}\"}"
fi

# check if "tomorrow-and-tomorrow-and-tomorrow" is in BOOK
BOOK=$(curl "http://localhost:8081/books/${BOOK_ID}")
if ! [[ $BOOK == *"${ISBN}"* ]]; then
    echo "'${ISBN}' not found in BOOK"
fi

# finally, delete the book
curl "http://localhost:8081/books/${BOOK_ID}" -X DELETE

# perform a curl, verify we get a 404 not found
if [[ $(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8081/books/${BOOK_ID}") -ne 404 ]]; then
    echo "Book not deleted"
fi

echo "Success! Test Passed"

