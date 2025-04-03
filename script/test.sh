#!/bin/bash

set -e

if [[ -z "$1" ]]; then
  echo "Error: REQUEST_LIMIT argument is required"
  exit 1
fi

MAX_REQUESTS=$1
TARGET_URL="http://traefik/whoami" # хост имя сервиса в кубе
TIMEOUT=1

echo "Starting rate limiter test..."
echo "Testing endpoint: $TARGET_URL"

status_codes=()

for i in $(seq 1 $((MAX_REQUESTS + 1))); do
    status_code=$(curl -s -o /dev/null -w "%{http_code}" -m $TIMEOUT "$TARGET_URL")
    status_codes+=($status_code)
    echo "Request $i: $status_code"
done

count_429=$(echo "${status_codes[@]}" | grep -o '429' | wc -l)

if [ "$count_429" -eq 1 ]; then
    echo "Test passed! Exactly one 429 response was received."
else
    echo "Test failed! Expected exactly one 429 response, but got $count_429."
    exit 1
fi
