#!/bin/bash

# Configuration
BASE_URL=${MLC_BASE_URL:-"http://localhost:9142"}
MODEL=${MLC_TEST_MODEL:-"multilingual-e5-small"}

echo "--- Testing mlc-localembed API at $BASE_URL ---"

# Function to check if server is up
check_server() {
    curl -s "$BASE_URL/api/health" > /dev/null
    return $?
}

if ! check_server; then
    echo "ERROR: Server not running at $BASE_URL"
    exit 1
fi

# 1. Health Check
echo -n "1. Health Check: "
status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/health")
if [ "$status" == "200" ]; then
    echo "OK"
else
    echo "FAILED (Status: $status)"
fi

# 2. List Tags
echo -n "2. List Tags: "
tags=$(curl -s "$BASE_URL/api/tags")
if [[ $tags == *"$MODEL"* ]]; then
    echo "OK (Found $MODEL)"
else
    echo "FAILED (Model $MODEL not found in tags)"
    echo "Response: $tags"
fi

# 3. Embed (Faker)
echo -n "3. Embed Faker: "
faker=$(curl -s -X POST "$BASE_URL/api/embed/faker" \
    -H "Content-Type: application/json" \
    -d "{\"model\": \"$MODEL\", \"input\": \"Hello World\"}")
if [[ $faker == *"embeddings"* ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "Response: $faker"
fi

# 4. Real Embed ($MODEL):
echo -n "4. Real Embed ($MODEL): "
embed=$(curl -s -X POST "$BASE_URL/api/embed" \
    -H "Content-Type: application/json" \
    -d "{\"model\": \"$MODEL\", \"input\": \"This is a real test.\"}")
if [[ $embed == *"embeddings"* ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "Response: $embed"
fi

# 5. Array Embed
echo -n "5. Array Embed: "
array_embed=$(curl -s -X POST "$BASE_URL/api/embed" \
    -H "Content-Type: application/json" \
    -d "{\"model\": \"$MODEL\", \"input\": [\"Hello\", \"World\"]}")
if [[ $array_embed == *"embeddings"* ]]; then
    if [[ $array_embed == *"["* ]]; then
        echo "OK (Received embeddings)"
    else
        echo "FAILED (Unexpected response structure)"
        echo "Response: $array_embed"
    fi
else
    echo "FAILED"
    echo "Response: $array_embed"
fi

echo "--- Tests completed ---"
