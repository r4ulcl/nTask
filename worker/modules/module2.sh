#!/bin/bash
echo Arguments: "$@"

SLEEP=$((5 + RANDOM % 10))
echo Sleep: $SLEEP
sleep $SLEEP
echo End