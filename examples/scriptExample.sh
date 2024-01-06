#!/bin/bash

# Function to send a POST request and get the task ID
function send_post_request() {
    local url="$1"
    local oauthToken="$2"
    local data="$3"
    
    # Send POST request and capture the task ID
    task_id=$(curl -s -k -X POST -H "Authorization: $oauthToken" -H "Content-Type: application/json" -d "$data" "$url" | jq -r '.id')
    echo "$task_id"
}

# Function to check the status of a task using a GET request
function get_task() {
    local url="$1"
    local oauthToken="$2"
    
    # Send GET request to check task status
    task=$(curl -s -k -H "Authorization: $oauthToken" "$url")
    echo "$task"
}

function wait_task(){
    local url="$1"
    local oauthToken="$2"
    local task_id="$3"
    # Wait for task done
    while true; do
        task=$(get_task "$url/task/$task_id" "$oauthToken" )
        status=$(echo $task | jq -r '.status')
        # Check if the status is not equal to "working"
        if [ "$status" == "done" ]; then
            #echo "Task completed successfully. Status: $status"
            
            echo $task
            break  # Exit the loop
        else
            #echo "Task still in progress. Status: $status"
            sleep 1  # Adjust the sleep duration as needed
        fi
    done
}

function wait_tasks(){
    local url="$1"
    local oauthToken="$2"
    shift 2 
    local task_ids=("$@")

    output_array=()
    pids=()

    echo -n '{"result":['

    # Loop ids
    array_length=${#task_ids[@]}
    for ((i=0; i<array_length; i++)); do
        task="${task_ids[i]}"

        #for task in "${task_ids[@]}"; do
        # execute in background
        wait_task $url $oauthToken $task &
        pids[$i]=$!
        # Remove from array
        task_ids=("${task_ids[@]/$task}")

    done

    # Wait for all background processes to finish
    for pid in "${pids[@]}"; do
        wait $pid
    done

    echo ']}'

}

# Define vars
## oauth token, IP, port, range to scan
oauthToken="WLJ2xVQZ5TXVw4qEznZDnmEE2"
nTaskIP="127.0.0.1"
nTaskPort="8080"
scanRange="127.0.0.1/28"
url="https://$nTaskIP:$nTaskPort"
# Array to store task IDs
task_ids=()

# Connect to Manager
## Send nmap ping only range
command="{\"module\": \"curl\", \"args\": \"-s ipinfo.io/city\"}"
task_data="{\"command\": [$command],\"priority\": 0}"
task_id=$(send_post_request "$url/task" "$oauthToken" "$task_data")
# add task_id to array
task_ids+=("$task_id")
echo task_ids: $task_ids

for task in "${task_ids[@]}"; do
    echo "$task"
done

# Wait for task done
OUTPUTS=($(wait_tasks $url $oauthToken "${task_ids[@]}"))

# Fix jq errors, no comma in json
AUX=${OUTPUTS[@]}
REPLACED='} {'
REPLACE='}, {'
FINAL=${AUX//$REPLACED/$REPLACE}


echo "NMAP results:"
#echo "${OUTPUTS[@]}" | jq
echo "${FINAL}" | jq

exit 0
