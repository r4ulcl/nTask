package utils

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
)

// VerifyWorkersLoop checks and sets if the workers are UP infinitely.
func VerifyWorkersLoop(db *sql.DB, config *ManagerConfig, verbose bool) {
	for {
		go verifyWorkers(db, config, verbose)
		time.Sleep(5 * time.Second)
	}
}

// verifyWorkers checks and sets if the workers are UP.
func verifyWorkers(db *sql.DB, config *ManagerConfig, verbose bool) {
	// Get all UP workers from the database
	workers, err := database.GetWorkerUP(db, verbose)
	if err != nil {
		log.Print(err)
	}

	// Verify each worker
	for _, worker := range workers {
		err := verifyWorker(db, config, &worker, verbose)
		if err != nil {
			log.Print(err)
		}
	}
}

// verifyWorker checks and sets if the worker is UP.
func verifyWorker(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose bool) error {
	var workerURL string
	if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			workerURL = "https://" + worker.IP + ":" + worker.Port + "/status"
		} else {
			workerURL = "http://" + worker.IP + ":" + worker.Port + "/status"
		}
	} else {
		workerURL = "http://" + worker.IP + ":" + worker.Port + "/status"
	}
	if verbose {
		log.Println("workerURL:", workerURL)
	}
	// Create an HTTP client and send a GET request to workerURL/status

	req, err := http.NewRequest("GET", workerURL, nil)
	if err != nil {
		if verbose {
			log.Println("Failed to create request to:", workerURL, " error:", err)
			log.Println("Delete worker:", worker.Name)
		}

		// If there is an error in creating the request, delete the worker from the database
		err := database.RmWorkerName(db, worker.Name, verbose)
		if err != nil {
			return err
		}
		return err
	}

	req.Header.Set("Authorization", worker.OauthToken)

	resp, err := config.ClientHTTP.Do(req)
	if err != nil {
		if verbose {
			log.Println("Error making request:", err)
		}
		// If there is an error in making the request, assume worker is offline
		count, err := database.GetWorkerDownCount(db, worker, verbose)
		if err != nil {
			return err
		}
		if count >= 3 {
			// If worker has been offline for 3 or more cycles, set it as offline in database
			err = database.SetWorkerUPto(false, db, worker, verbose)
			if err != nil {
				return err
			}
			// Reset the count to 0
			err = database.SetWorkerDownCount(0, db, worker, verbose)
			if err != nil {
				return err
			}

			// Set as 'failed' all workers tasks
			err = database.SetTasksWorkerFailed(db, worker.Name, verbose)
			if err != nil {
				return err
			}
		} else {
			// If worker has been offline for less than 3 cycles, increment the count
			err = database.AddWorkerDownCount(db, worker, verbose)
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
	err = database.SetWorkerUPto(true, db, worker, verbose)
	if err != nil {
		return err
	}

	// Read the response body into a byte slice
	body, err := io.ReadAll(resp.Body)
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
		err := database.SetWorkerworkingTo(status.IddleThreads, db, worker.Name, verbose)
		if err != nil {
			return err
		}
	}

	return nil

}

// SendAddTask sends a request to a worker to add a task.
func SendAddTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose bool) error {
	var workerURL string
	if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			workerURL = "https://" + worker.IP + ":" + worker.Port + "/task"
		} else {
			workerURL = "http://" + worker.IP + ":" + worker.Port + "/task"
		}
	} else {
		workerURL = "http://" + worker.IP + ":" + worker.Port + "/task"
	}

	// Set task as executed
	err := database.SetTaskExecutedAt(db, task.ID, verbose)
	if err != nil {
		log.Println("Error SetWorkerNameTask in request:", err)
		return err
	}

	// Set workerName in DB and in object
	task.WorkerName = worker.Name
	err = database.SetTaskWorkerName(db, task.ID, worker.Name, verbose)
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
	req, err := http.NewRequest("POST", workerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request:", err)
		return err
	}

	// Add Authorization header
	req.Header.Set("Authorization", worker.OauthToken)

	// Specify the content type as JSON
	req.Header.Set("Content-Type", "application/json")

	// Send the request

	resp, err := config.ClientHTTP.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		if verbose {
			log.Println("POST request was successful")
		}
		// Set the task and worker as working
		err := database.SetTaskStatus(db, task.ID, "running", verbose)
		if err != nil {
			return err
		}

		//Add 1 to working worker
		err = database.SubtractWorkerIddleThreads1(db, worker.Name, verbose)
		if err != nil {
			return err
		}
	} else {
		message := "POST request failed with status:" + resp.Status + ". worker problably working"
		return fmt.Errorf(message)
	}

	return nil
}

// SendDeleteTask sends a request to a worker to stop and delete a task.
func SendDeleteTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose bool) error {
	var workerURL string
	if transport, ok := config.ClientHTTP.Transport.(*http.Transport); ok {
		if transport.TLSClientConfig != nil {
			workerURL = "https://" + worker.IP + ":" + worker.Port + "/task/" + task.ID
		} else {
			workerURL = "http://" + worker.IP + ":" + worker.Port + "/task/" + task.ID
		}
	} else {
		workerURL = "http://" + worker.IP + ":" + worker.Port + "/task/" + task.ID
	}

	// Create a new DELETE request
	req, err := http.NewRequest("DELETE", workerURL, nil)
	if err != nil {
		return err
	}

	// Add Authorization header
	req.Header.Set("Authorization", worker.OauthToken)

	// Specify the content type as JSON
	req.Header.Set("Content-Type", "application/json")

	// Send the request

	resp, err := config.ClientHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		if verbose {
			log.Println("POST request was successful")
		}
		// Set the task and worker as not working
		err := database.SetTaskStatus(db, task.ID, "deleted", verbose)
		if err != nil {
			return err
		}
		err = database.SubtractWorkerIddleThreads1(db, worker.Name, verbose)
		if err != nil {
			return err
		}
	} else {
		message := "POST request failed with status:" + resp.Status + ". worker problably working"
		return fmt.Errorf(message)
	}

	return nil
}

/*
// SendGetTask sends a request to a worker to get the status of a task.
func SendGetTask(db *sql.DB, OauthTokenWorkers string, worker *globalstructs.Worker, task globalstructs.Task, verbose bool) (globalstructs.Task, error) {
	return task, nil
}
*/

// CreateTLSClientWithCACert from cert.pem
func CreateTLSClientWithCACert(caCertPath string, verifyAltName, verbose bool) (*http.Client, error) {
	// Load CA certificate from file
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		fmt.Printf("Failed to read CA certificate file: %v\n", err)
		return nil, err
	}

	// Create a certificate pool and add the CA certificate
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	// Replace 'cert' with the expected certificate that the server should present
	//var cert *x509.Certificate

	var tlsConfig *tls.Config

	// Create a TLS configuration with the custom VerifyPeerCertificate function
	if !verifyAltName {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // Enable server verification
			RootCAs:            certPool,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				if len(rawCerts) == 0 {
					return fmt.Errorf("no certificates provided by the server")
				}

				serverCert, err := x509.ParseCertificate(rawCerts[0])
				if err != nil {
					return fmt.Errorf("failed to parse server certificate: %v", err)
				}

				// Verify the server certificate against the CA certificate
				opts := x509.VerifyOptions{
					Roots:         certPool,
					Intermediates: x509.NewCertPool(),
				}
				_, err = serverCert.Verify(opts)
				if err != nil {
					return fmt.Errorf("failed to verify server certificate: %v", err)
				}

				return nil
			},
		}
	} else {
		log.Println("verifyAltName YES", !verifyAltName)

		tlsConfig = &tls.Config{
			InsecureSkipVerify: false, // Ensure that server verification is enabled
			RootCAs:            certPool,
		}
	}

	// Create HTTP client with TLS
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return client, nil
}
