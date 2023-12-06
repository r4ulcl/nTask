package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

// AddWorker sends a POST request to add a worker to the manager
func AddWorker(config *WorkerConfig, verbose bool) error {
	// Create a Worker object with the provided configuration
	worker := globalstructs.Worker{
		Name:         config.Name,
		Port:         config.Port,
		OauthToken:   config.OAuthToken,
		IddleThreads: config.IddleThreads,
		UP:           true,
	}

	// Marshal the worker object into JSON
	payload, _ := json.Marshal(worker)

	// Create a new POST request to add the worker
	url := "http://" + config.ManagerIP + ":" + config.ManagerPort + "/worker"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	// Add custom headers, including the OAuth header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.ManagerOauthToken)

	// Create an HTTP client and make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check if the response status code is not 200
	if resp.StatusCode != 200 {
		return fmt.Errorf("error adding the worker %s", body)
	}

	return nil
}

// CallbackTaskMessage sends a POST request to the manager to callback with a task message
func CallbackTaskMessage(config *WorkerConfig, task *globalstructs.Task, verbose bool) error {
	// Create the callback URL using the manager IP and port
	url := "http://" + config.ManagerIP + ":" + config.ManagerPort + "/callback"

	// Marshal the task object into JSON
	payload, _ := json.Marshal(task)

	// Create a new POST request to send the task message
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Error creating request:", err)
		return err
	}

	// Add custom headers, including the OAuth header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.ManagerOauthToken)

	// Create an HTTP client and make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		return err
	}
	defer resp.Body.Close()

	log.Println("Status Code:", resp.Status)
	// Handle the response body as needed
	return nil
}
