#!/bin/bash

# Usage: ./script.sh <num_requests>

NUM_REQUESTS=$1

if [ -z "$NUM_REQUESTS" ]; then
  echo "Usage: $0 <num_requests>"
  exit 1
fi

URL="http://localhost:8080/api/hello"
HEADER="X-Caller-Service: a10nsp"

for ((i=1; i<=NUM_REQUESTS; i++))
do
  echo "Request #$i"

  curl -s -H "$HEADER" "$URL"

  # Random sleep between 0–10 seconds
  SLEEP_TIME=$((RANDOM % 5))
  echo "Sleeping for $SLEEP_TIME seconds..."
  sleep $SLEEP_TIME
done

echo "Done."
