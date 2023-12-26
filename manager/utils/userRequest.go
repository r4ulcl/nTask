package utils

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// CallbackUserTaskMessage is a function that sends a task message as a callback to a specified URL
func CallbackUserTaskMessage(config *ManagerConfig, task *globalstructs.Task, verbose, debug bool) {
	url := task.CallbackURL

	// Convert the task to a JSON payload
	payload, _ := json.Marshal(task)

	// Create a new request with the POST method and the payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	// Add custom headers, including the OAuth header
	req.Header.Set("Content-Type", "application/json")
	if task.CallbackToken != "" {
		req.Header.Set("Authorization", task.CallbackToken)
	}

	// Create an HTTP client and make the request
	resp, err := config.ClientHTTP.Do(req)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return
	}
	defer resp.Body.Close()

	if debug {
		log.Println("Status Code:", resp.Status)
	}
	// Handle the response body as needed
}
