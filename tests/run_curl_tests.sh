#!/bin/bash

# Configuration
BASE_URL=${MLC_BASE_URL:-"http://localhost:9142"}
MODEL=${MLC_TEST_MODEL:-"multilingual-e5-small"}

echo "--- Testing mlc-localembed API at $BASE_URL ---"

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
faker=$(curl -s -X POST "$BASE_URL/api/embed/faker" 
    -H "Content-Type: application/json" 
    -d "{"model": "$MODEL", "input": "Hello World"}")
if [[ $faker == *"embeddings"* ]]; then
    echo "OK"
else
    echo "FAILED"
    echo "Response: $faker"
fi

# 4. Real Embed (requires model loaded)
echo -n "4. Real Embed ($MODEL): "
embed=$(curl -s -X POST "$BASE_URL/api/embed" 
    -H "Content-Type: application/json" 
    -d "{"model": "$MODEL", "input": "This is a real test."}")
if [[ $embed == *"embeddings"* ]]; then
    echo "OK"
else
    echo "FAILED (Maybe model not downloaded?)"
    echo "Response: $embed"
fi

echo "--- Tests completed ---"
