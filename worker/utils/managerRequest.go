package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
)

func AddWorker(config *WorkerConfig) error {
	worker := globalStructs.Worker{
		Name:       config.Name,
		Port:       config.Port,
		OauthToken: config.OAuthToken,
		Working:    false,
		UP:         true,
	}

	payload, _ := json.Marshal(worker)

	req, err := http.NewRequest("POST", "http:// "+config.ManagerIP+":"+config.ManagerPort+"/worker", bytes.NewBuffer(payload))
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

	// IF response is not 200 error!!
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error adding the worker %s", body)
	}

	return nil
}

func CallbackTaskMessage(config *WorkerConfig, task *globalStructs.Task) {
	url := "http:// " + config.ManagerIP + ":" + config.ManagerPort + "/callback"

	payload, _ := json.Marshal(task)

	// Create a new request with the POST method and the payload
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add custom headers, including the OAuth header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", config.ManagerOauthToken)

	// Create an HTTP client and make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status Code:", resp.Status)
	// Handle the response body as needed

}
