package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"log"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
	"github.com/r4ulcl/NetTask/manager/database"
)

// verifyWorkersLoop check and set if the workers are UP infinite
func VerifyWorkersLoop(db *sql.DB) {
	for {
		go verifyWorkers(db)
		time.Sleep(5 * time.Second)
	}
}

// verifyWorkers check and set if the workers are UP
func verifyWorkers(db *sql.DB) {

	workers, err := database.GetWorkerUP(db)
	if err != nil {
		log.Print(err)
	}

	for _, worker := range workers {
		err := verifyWorker(db, &worker)
		if err != nil {
			log.Print(err)
		}
	}
}

// VerifyWorker check and set if the workers are UP
func verifyWorker(db *sql.DB, worker *globalStructs.Worker) error {
	workerURL := "http:// " + worker.IP + ":" + worker.Port

	// Create an HTTP client and send a GET request
	client := &http.Client{}
	req, err := http.NewRequest("GET", workerURL+"/status", nil)
	if err != nil {
		fmt.Printf("Failed to create request to %s: %v\nDelete worker: %s", workerURL, err, worker.Name)
		// Incorrect DATA, delete worker
		err := database.RmWorkerName(db, worker.Name)
		if err != nil {
			return err
		}
		return err
	}

	req.Header.Set("Authorization", worker.OauthToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		// if error making request is offline!
		count, err := database.GetWorkerCount(db, worker)
		if err != nil {
			return err
		}
		if count >= 3 {
			err = database.SetWorkerUPto(false, db, worker)
			if err != nil {
				return err
			}
			err = database.SetWorkerCount(0, db, worker)
			if err != nil {
				return err
			}
		} else {
			err = database.AddWorkerCount(db, worker)
			if err != nil {
				return err
			}
		}
		return err
	}
	defer resp.Body.Close()

	// if no error making request is online!
	err = database.SetWorkerUPto(true, db, worker)
	if err != nil {
		return err
	}

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
		err := database.SetWorkerworkingTo(status.Working, db, worker)
		if err != nil {
			return err
		}
	}

	return nil

}

// SendAddTask send to a worker a request to add a task
func SendAddTask(db *sql.DB, worker *globalStructs.Worker, task *globalStructs.Task) error {
	workerURL := "http:// " + worker.IP + ":" + worker.Port

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
	req.Header.Set("Authorization", worker.OauthToken)

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
		// set worker and task working
		err := database.SetTaskStatus(db, task.ID, "running")
		if err != nil {
			return err
		}
		err = database.SetWorkerworkingTo(true, db, worker)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("POST request failed with status:", resp.Status)
	}

	return nil
}

// SendDeleteTask send request to a worker to stop and delete a task
func SendDeleteTask(db *sql.DB, worker *globalStructs.Worker, task *globalStructs.Task) error {
	workerURL := "http:// " + worker.IP + ":" + worker.Port + "/task/" + task.ID

	// Create a new DELETE request
	req, err := http.NewRequest("DELETE", workerURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Add Authorization header
	req.Header.Set("Authorization", worker.OauthToken)

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
		// set worker and task working
		err := database.SetTaskStatus(db, task.ID, "deleted")
		if err != nil {
			return err
		}
		err = database.SetWorkerworkingTo(false, db, worker)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("POST request failed with status:", resp.Status)
	}

	return nil
}

/*
// SendGetTask send request to a worker of a task status
func SendGetTask(db *sql.DB, OauthTokenWorkers string, worker *globalStructs.Worker, task globalStructs.Task) (globalStructs.Task, error) {
	return task, nil
}
*/
