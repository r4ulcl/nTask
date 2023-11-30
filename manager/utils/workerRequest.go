package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
	"github.com/r4ulcl/NetTask/manager/database"
)

//verifyWorkersLoop check and set if the workers are UP infinite
func VerifyWorkersLoop(db *sql.DB) {
	for {
		go verifyWorkers(db)
		time.Sleep(5 * time.Second)
	}
}

//verifyWorkers check and set if the workers are UP
func verifyWorkers(db *sql.DB) {

	workers, err := database.GetWorkerUP(db)
	if err != nil {
		fmt.Println(err)
	}
	for _, worker := range workers {
		verifyWorker(db, &worker)
	}

}

//VerifyWorker check and set if the workers are UP
func verifyWorker(db *sql.DB, worker *globalStructs.Worker) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port

	// Create an HTTP client and send a GET request
	client := &http.Client{}
	req, err := http.NewRequest("GET", workerURL+"/status", nil)
	if err != nil {
		fmt.Printf("Failed to create request to %s: %v\nDelete worker: %s", workerURL, err, worker.Name)
		//Incorrect DATA, delete worker
		database.RmWorkerName(db, worker.Name)
		return err
	}

	req.Header.Set("Authorization", worker.OauthToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		//if error making request is offline!
		count, err := database.GetWorkerCount(db, worker)
		if err != nil {
			return err
		}
		if count >= 3 {
			database.SetWorkerUPto(false, db, worker)
			database.SetWorkerCount(0, db, worker)
		} else {
			database.AddWorkerCount(db, worker)
		}
		return err
	}
	defer resp.Body.Close()

	//if no error making request is online!
	database.SetWorkerUPto(true, db, worker)

	// Read the response body into a byte slice
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	// Unmarshal the JSON into a TaskResponse struct
	var status globalStructs.WorkerStatus
	err = json.Unmarshal(body, &status)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	// If worker status is not the same as storage in DB update
	if status.Working != worker.Working {
		database.SetWorkerworkingTo(status.Working, db, worker)
	}

	return nil

}

//SendAddTask send to a worker a request to add a task
func SendAddTask(db *sql.DB, OauthTokenWorkers string, worker *globalStructs.Worker, task *globalStructs.Task) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port

	// Set workerName in DB and in object
	task.WorkerName = worker.Name
	err := database.SetTaskWorkerName(db, task.ID, worker.Name)
	if err != nil {
		fmt.Println("Error SetWorkerNameTask in request:", err)
		return err
	}

	// Convert struct to JSON
	fmt.Println(task.ID)
	jsonData, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Create a new POST request with JSON payload
	req, err := http.NewRequest("POST", workerURL+"/task", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Add Authorization header
	req.Header.Set("Authorization", OauthTokenWorkers)

	// Specify the content type as JSON
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		fmt.Println("POST request was successful")
		//set worker and task working
		database.SetTaskStatus(db, task.ID, "running")
		database.SetWorkerworkingTo(true, db, worker)
	} else {
		fmt.Println("POST request failed with status:", resp.Status)
	}

	return nil
}

//SendDeleteTask send request to a worker to stop and delete a task
func SendDeleteTask(db *sql.DB, OauthTokenWorkers string, worker *globalStructs.Worker, task globalStructs.Task) error {
	return nil
}

/*
//SendGetTask send request to a worker of a task status
func SendGetTask(db *sql.DB, OauthTokenWorkers string, worker *globalStructs.Worker, task globalStructs.Task) (globalStructs.Task, error) {
	return task, nil
}
*/
