package utils

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/manager/database"
)

// VerifyWorkersLoop checks and sets if the workers are UP infinitely.
func VerifyWorkersLoop(db *sql.DB) {
	for {
		go verifyWorkers(db)
		time.Sleep(5 * time.Second)
	}
}

// verifyWorkers checks and sets if the workers are UP.
func verifyWorkers(db *sql.DB) {
	// Get all UP workers from the database
	workers, err := database.GetWorkerUP(db)
	if err != nil {
		log.Print(err)
	}

	// Verify each worker
	for _, worker := range workers {
		err := verifyWorker(db, &worker)
		if err != nil {
			log.Print(err)
		}
	}
}

// verifyWorker checks and sets if the worker is UP.
func verifyWorker(db *sql.DB, worker *globalstructs.Worker) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port

	// Create an HTTP client and send a GET request to workerURL/status
	client := &http.Client{}
	req, err := http.NewRequest("GET", workerURL+"/status", nil)
	if err != nil {
		log.Println("Failed to create request to:", workerURL, " error:", err)
		log.Println("Delete worker:", worker.Name)

		// If there is an error in creating the request, delete the worker from the database
		err := database.RmWorkerName(db, worker.Name)
		if err != nil {
			return err
		}
		return err
	}

	req.Header.Set("Authorization", worker.OauthToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error making request:", err)
		// If there is an error in making the request, assume worker is offline
		count, err := database.GetWorkerDownCount(db, worker)
		if err != nil {
			return err
		}
		if count >= 3 {
			// If worker has been offline for 3 or more cycles, set it as offline in database
			err = database.SetWorkerUPto(false, db, worker)
			if err != nil {
				return err
			}
			// Reset the count to 0
			err = database.SetWorkerDownCount(0, db, worker)
			if err != nil {
				return err
			}

			// Set as 'failed' all workers tasks
			err = database.SetTasksWorkerFailed(db, worker.Name)
			if err != nil {
				return err
			}
		} else {
			// If worker has been offline for less than 3 cycles, increment the count
			err = database.AddWorkerDownCount(db, worker)
			if err != nil {
				return err
			}
		}
		return err
	}
	defer resp.Body.Close()

	// if response is not 200 error
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Unexpected status code %d\n", resp.StatusCode)
		return fmt.Errorf("error:", resp.Status)
	}
	// If there is no error in making the request, assume worker is online
	err = database.SetWorkerUPto(true, db, worker)
	if err != nil {
		return err
	}

	// Read the response body into a byte slice
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return err
	}

	// Unmarshal the JSON into a TaskResponse struct
	var status globalstructs.WorkerStatus
	err = json.Unmarshal(body, &status)
	if err != nil {
		log.Println("Error unmarshalling JSON:", err, body)
		return err
	}

	// If worker status is not the same as stored in the DB, update the DB
	if status.IddleThreads != worker.IddleThreads {
		err := database.SetWorkerworkingTo(status.IddleThreads, db, worker.Name)
		if err != nil {
			return err
		}
	}

	return nil

}

// SendAddTask sends a request to a worker to add a task.
func SendAddTask(db *sql.DB, worker *globalstructs.Worker, task *globalstructs.Task) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port

	// Set workerName in DB and in object
	task.WorkerName = worker.Name
	err := database.SetTaskWorkerName(db, task.ID, worker.Name)
	if err != nil {
		log.Println("Error SetWorkerNameTask in request:", err)
		return err
	}

	// Convert the struct to JSON
	jsonData, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Create a new POST request with JSON payload
	req, err := http.NewRequest("POST", workerURL+"/task", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request:", err)
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
		log.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		log.Println("POST request was successful")
		// Set the task and worker as working
		err := database.SetTaskStatus(db, task.ID, "running")
		if err != nil {
			return err
		}

		//Add 1 to working worker
		err = database.SubtractWorkerIddleThreads1(db, worker.Name)
		if err != nil {
			return err
		}
	} else {
		log.Println("POST request failed with status:", resp.Status)
	}

	return nil
}

// SendDeleteTask sends a request to a worker to stop and delete a task.
func SendDeleteTask(db *sql.DB, worker *globalstructs.Worker, task *globalstructs.Task) error {
	workerURL := "http://" + worker.IP + ":" + worker.Port + "/task/" + task.ID

	// Create a new DELETE request
	req, err := http.NewRequest("DELETE", workerURL, nil)
	if err != nil {
		log.Println("Error creating request:", err)
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
		log.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		log.Println("POST request was successful")
		// Set the task and worker as not working
		err := database.SetTaskStatus(db, task.ID, "deleted")
		if err != nil {
			return err
		}
		err = database.SubtractWorkerIddleThreads1(db, worker.Name)
		if err != nil {
			return err
		}
	} else {
		log.Println("POST request failed with status:", resp.Status)
	}

	return nil
}

/*
// SendGetTask sends a request to a worker to get the status of a task.
func SendGetTask(db *sql.DB, OauthTokenWorkers string, worker *globalstructs.Worker, task globalstructs.Task) (globalstructs.Task, error) {
	return task, nil
}
*/
