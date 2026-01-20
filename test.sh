#!/usr/bin/env bash
# infinite_curl.sh  –  endlessly enqueue “sleep X” tasks with X ∈ [1,20]

AUTH="WLJ2xVQZ5TXVw4qEznZDnmEEV"
URL="http://127.0.0.1:8080/task"

MAX_ITER=10000

for ((i=1; i<=MAX_ITER; i++)); do
  # Pick a random integer 1-20
  X=$(( RANDOM % 60 + 1 ))

  # Compose the JSON payload with that X
  read -r -d '' payload <<EOF
{
  "commands": [
    {
      "args": "sleep $X ; date",
      "module": "exec"
    }
  ],
  "name": "sleep $X $i",
  "notes": "string",
  "priority": $X,
  "timeout": 0
}
EOF

  # Fire the request
  curl -s -X POST "$URL" \
       -H 'accept: application/json' \
       -H "Authorization: $AUTH" \
       -H 'Content-Type: application/json' \
       -d "$payload"   > /dev/null &

  #echo        # newline after each response
  #sleep 0     # brief pause so you don’t hammer the endpoint too hard
done

wait 
