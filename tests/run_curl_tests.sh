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

# 6. Similarity API
echo -n "6. Similarity API: "
sim_resp=$(curl -s -X POST "$BASE_URL/api/test/similarity" \
    -H "Content-Type: application/json" \
    -d "{
        \"model\": \"$MODEL\", 
        \"query\": \"Which city is the capital of France?\", 
        \"documents\": [\"Paris is the capital.\", \"Berlin is in Germany.\", \"Die Sonne ist ein Stern.\"]
    }")
if [[ $sim_resp == *"scores"* ]]; then
    # Check if we have 3 results by counting "document" occurrences
    count=$(echo "$sim_resp" | grep -o "\"document\"" | wc -l)
    if [ "$count" -eq "3" ]; then
        echo "OK (Received 3 results)"
    else
        echo "FAILED (Expected 3 results, got $count)"
        echo "Response: $sim_resp"
    fi
else
    echo "FAILED"
    echo "Response: $sim_resp"
fi

# 7. Stats API
echo -n "7. Stats API: "
stats_resp=$(curl -s "$BASE_URL/api/stats")
if [[ $stats_resp == *"total_requests"* ]]; then
    uptime=$(echo $stats_resp | grep -o "\"uptime_seconds\":[0-9]*" | cut -d: -f2)
    echo "OK (Uptime: ${uptime}s)"
else
    echo "FAILED"
    echo "Response: $stats_resp"
fi

# 8. Model Handling Test:
echo "8. Model Handling Test:"
MODEL_DISABLED="BAAI/bge-small-en-v1.5"
echo -n "   - Requesting enabled model ($MODEL): "
curl -s -X POST "$BASE_URL/api/embed" -H "Content-Type: application/json" -d "{\"model\": \"$MODEL\", \"input\": \"test\"}" | grep -q "embeddings" && echo "OK" || echo "FAILED"

echo -n "   - Requesting disabled model ($MODEL_DISABLED): "
resp_disabled=$(curl -s -X POST "$BASE_URL/api/embed" -H "Content-Type: application/json" -d "{\"model\": \"$MODEL_DISABLED\", \"input\": \"test\"}")
if [[ $resp_disabled == *"disabled"* ]] || [[ $resp_disabled == *"not found"* ]]; then
    echo "OK (Correctly rejected)"
else
    echo "FAILED (Should have been rejected as disabled)"
    echo "Response: $resp_disabled"
fi

# 9. Error Handling Tests
echo "9. Error Handling Tests:"

echo -n "   - Empty input: "
empty_resp=$(curl -s -X POST "$BASE_URL/api/embed" -H "Content-Type: application/json" -d "{\"model\": \"$MODEL\", \"input\": \"\"}")
if [[ $empty_resp == *"empty"* ]]; then echo "OK"; else echo "FAILED ($empty_resp)"; fi

echo -n "   - Very long input (Token limit): "
# Create a string that is definitely longer than 512 tokens (approx 2000 chars)
LONG_TEXT=$(printf 'word %.0s' {1..1000})
limit_resp=$(curl -s -X POST "$BASE_URL/api/embed" -H "Content-Type: application/json" -d "{\"model\": \"$MODEL\", \"input\": \"$LONG_TEXT\"}")
if [[ $limit_resp == *"token limit"* ]]; then
    echo "OK (Correctly detected limit)"
else
    echo "FAILED (Expected token limit error)"
    echo "Response: $limit_resp"
fi

# 10. Parallel Request Test (Multi-user simulation)
echo "10. Parallel Request Test (Simulating 5 concurrent users):"
for i in {1..5}; do
    (
        resp=$(curl -s -X POST "$BASE_URL/api/embed" -H "Content-Type: application/json" -d "{\"model\": \"$MODEL\", \"input\": \"Parallel test request $i\"}")
        if [[ $resp == *"embeddings"* ]]; then
            echo "   [User $i] SUCCESS"
        else
            echo "   [User $i] FAILED: $resp"
        fi
    ) &
done
wait
echo "   Parallel tests finished."

echo "--- Tests completed ---"
