#!/bin/bash

# Usage: ./script.sh <num_requests_per_thread>

NUM_REQUESTS=$1
THREADS=10

if [ -z "$NUM_REQUESTS" ]; then
echo "Usage: $0 <num_requests_per_thread>"
exit 1
fi

URL="http://localhost:8080/api/hello"

# List of caller services

SERVICES=("a10nsp" "olt" "holt" "spine" "leaf" "DPU")

worker() {
local thread_id=$1

# Pick service based on thread (round-robin)

local service_index=$(( (thread_id - 1) % ${#SERVICES[@]} ))
local service_name=${SERVICES[$service_index]}

local header="X-Caller-Service: $service_name"

for ((i=1; i<=NUM_REQUESTS; i++))
do
echo "[Thread $thread_id][$service_name] Request #$i"

```
response=$(curl -s -w "%{http_code}" -o /dev/null -H "$header" "$URL")
echo "[Thread $thread_id][$service_name] Status: $response"

# Random sleep between 0–10 seconds
SLEEP_TIME=$((RANDOM % 11))
echo "[Thread $thread_id][$service_name] Sleeping for $SLEEP_TIME sec..."
sleep $SLEEP_TIME
```

done

echo "[Thread $thread_id][$service_name] Done."
}

# Start threads

for ((t=1; t<=THREADS; t++))
do
worker $t &
done

# Wait for all threads to finish

wait

echo "All threads completed."

