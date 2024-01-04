package utils

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
)

// VerifyWorkersLoop checks and sets if the workers are UP infinitely.
func VerifyWorkersLoop(db *sql.DB, config *ManagerConfig, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	for {
		go verifyWorkers(db, config, verbose, debug, wg, writeLock)
		time.Sleep(5 * time.Second)
	}
}

// verifyWorkers checks and sets if the workers are UP.
func verifyWorkers(db *sql.DB, config *ManagerConfig, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	// Get all UP workers from the database
	workers, err := database.GetWorkers(db, verbose, debug)
	if err != nil {
		log.Print("GetWorkerUP", err)
	}

	// Verify each worker
	for _, worker := range workers {
		err := verifyWorker(db, config, &worker, verbose, debug, wg, writeLock)
		if err != nil {
			log.Print("verifyWorker ", err)
		}
	}
}

func SendMessage(conn *websocket.Conn, message []byte, verbose, debug bool, writeLock *sync.Mutex) error {
	writeLock.Lock()
	defer writeLock.Unlock()
	if debug {
		log.Println("SendMessage", string(message))
	}
	// check if websocket alive
	err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
	if err != nil {
		log.Println("Error in websocket", string(message))
		return err
	}

	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}
	if debug {
		log.Println("SendMessage OK", string(message))
	}
	return nil
}

// verifyWorker checks and sets if the worker is UP.
func verifyWorker(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	if debug {
		log.Println("verifyWorker", worker.Name)
	}
	conn := config.WebSockets[worker.Name]
	if conn == nil {
		if debug {
			log.Println("Error: The worker doesnt have a websocket", worker.Name)
		}

		delete(config.WebSockets, worker.Name)

		err := database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
		if err != nil {
			return err
		}

		// Set as 'pending' all workers tasks to REDO
		err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
		if err != nil {
			return err
		}
		return nil
	}

	msg := globalstructs.WebsocketMessage{
		Type: "status",
		Json: "{}",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		if debug {
			log.Println("Error: json.Marshal(msg):", err)
		}
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg, writeLock)
		if err != nil {
			return err
		}
		return err
	}

	/*

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
		if debug {
			log.Println("workerURL:", workerURL)
		}
		// Create an HTTP client and send a GET request to workerURL/status

		req, err := http.NewRequest("GET", workerURL, nil)
		if err != nil {
			if debug {
				log.Println("Failed to create request to:", workerURL, " error:", err)
				log.Println("Delete worker:", worker.Name)
			}

			// If there is an error in creating the request, delete the worker from the database
			err := database.RmWorkerName(db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}
			return err
		}

		req.Header.Set("Authorization", worker.OauthToken)

		resp, err := config.ClientHTTP.Do(req)
		if err != nil {
			if debug {
				log.Println("Error making request:", err)
			}
			// If there is an error in making the request, assume worker is offline
			count, err := database.GetWorkerDownCount(db, worker, verbose, debug)
			if err != nil {
				return err
			}
			if count >= 3 {
				// If worker has been offline for 3 or more cycles, set it as offline in database
				err = database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
				if err != nil {
					return err
				}
				// Reset the count to 0
				err = database.SetWorkerDownCount(0, db, worker, verbose, debug, wg)
				if err != nil {
					return err
				}

				// Set as 'pending' all workers tasks to REDO
				err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
				if err != nil {
					return err
				}
			} else {
				// If worker has been offline for less than 3 cycles, increment the count
				err = database.AddWorkerDownCount(db, worker, verbose, debug, wg)
				if err != nil {
					return err
				}
			}
			return err
		}
		defer resp.Body.Close()

		// if response is not 200 error
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Error: Unexpected status code:", resp.Status)
		}
		// If there is no error in making the request, assume worker is online
		err = database.SetWorkerUPto(true, db, worker, verbose, debug, wg)
		if err != nil {
			return err
		}

		// Read the response body into a byte slice
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Error reading response body:", err)
		}

		// Unmarshal the JSON into a TaskResponse struct
		var status globalstructs.WorkerStatus
		err = json.Unmarshal(body, &status)
		if err != nil {
			return fmt.Errorf("Error unmarshalling JSON:", err)
		}

		// If worker status is not the same as stored in the DB, update the DB
		if status.IddleThreads != worker.IddleThreads {
			err := database.SetIddleThreadsTo(status.IddleThreads, db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}
		}*/

	return nil

}

// SendAddTask sends a request to a worker to add a task.
func SendAddTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	log.Println("SendAddTask")
	//Sustract 1 Iddle Thread in worker
	err := database.SubtractWorkerIddleThreads1(db, worker.Name, verbose, debug, wg)
	if err != nil {
		return err
	}
	// add 1 on callback

	conn := config.WebSockets[worker.Name]
	if conn == nil {
		/*delete(config.WebSockets, worker.Name)

		err := database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
		if err != nil {
			return err
		}

		// Set as 'pending' all workers tasks to REDO
		err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
		if err != nil {
			return err
		}*/
		return fmt.Errorf("Error, websocket not found")
	}

	// Set workerName in DB and in object
	task.WorkerName = worker.Name

	// Tast to json
	// Convert the struct to JSON
	jsonDataTask, err := json.Marshal(task)
	if err != nil {
		return err
	}

	msg := globalstructs.WebsocketMessage{
		Type: "addTask",
		Json: string(jsonDataTask),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg, writeLock)
		if err != nil {
			return err
		}
		return err
	}

	// Set task as executed
	err = database.SetTaskExecutedAtNow(db, task.ID, verbose, debug, wg)
	if err != nil {
		return fmt.Errorf("Error SetTaskExecutedAt in request:", err)
	}

	// Set workerName in DB and in object
	err = database.SetTaskWorkerName(db, task.ID, worker.Name, verbose, debug, wg)
	if err != nil {
		return fmt.Errorf("Error SetWorkerNameTask in request:", err)
	}

	if verbose {
		log.Println("Task send successfully")
	}

	/*/ --------------------------------------------------------------------------------------------------------

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

	// Set workerName in DB and in object
	task.WorkerName = worker.Name

	// Convert the struct to JSON
	jsonData, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Create a new POST request with JSON payload
	req, err := http.NewRequest("POST", workerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Error creating request:", err)
	}

	// Add Authorization header
	req.Header.Set("Authorization", worker.OauthToken)

	// Specify the content type as JSON
	req.Header.Set("Content-Type", "application/json")

	// Get task current executedAt
	oldExecutedAt, err := database.GetTaskExecutedAt(db, task.ID, verbose, debug)
	if err != nil {
		return fmt.Errorf("Error GetTaskExecutedAt in request:", err)
	}

	// Set the task as running if its pending
	err = database.SetTaskStatusIfPending(db, task.ID, "running", verbose, debug, wg)
	if err != nil {
		return err
	}

	// Send the request

	resp, err := config.ClientHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request:", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		if debug {
			log.Println("POST request was successful")
		}

		if verbose {
			log.Println("Send Task", task.ID, "to worker", worker.Name)
		}

		// Set task as executed
		err = database.SetTaskExecutedAtNow(db, task.ID, verbose, debug, wg)
		if err != nil {
			return fmt.Errorf("Error SetTaskExecutedAt in request:", err)
		}

		// Set workerName in DB and in object
		err = database.SetTaskWorkerName(db, task.ID, worker.Name, verbose, debug, wg)
		if err != nil {
			return fmt.Errorf("Error SetWorkerNameTask in request:", err)
		}

		if verbose {
			log.Println("Task send successfully")
		}

	} else {
		// if comunication error

		// Set task as executed at old
		err = database.SetTaskExecutedAt(oldExecutedAt, db, task.ID, verbose, debug, wg)
		if err != nil {
			return fmt.Errorf("Error SetTaskExecutedAt in request:", err)
		}

		// Set the task as pending again
		err = database.SetTaskStatus(db, task.ID, "pending", verbose, debug, wg)
		if err != nil {
			return err
		}

		if resp.StatusCode == 423 {
			message := "POST request failed with status: 423. Worker already working"
			return fmt.Errorf(message)
		} else {
			return fmt.Errorf("POST request failed with status:", resp.Status)

		}
	}
	*/
	return nil
}

// SendDeleteTask sends a request to a worker to stop and delete a task.
func SendDeleteTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	conn := config.WebSockets[worker.Name]
	if conn == nil {
		/*delete(config.WebSockets, worker.Name)

		err := database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
		if err != nil {
			return err
		}

		// Set as 'pending' all workers tasks to REDO
		err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
		if err != nil {
			return err
		}*/
		return fmt.Errorf("Error, websocket not found")
	}

	// Tast to json
	// Convert the struct to JSON
	jsonDataTask, err := json.Marshal(task)
	if err != nil {
		return err
	}

	msg := globalstructs.WebsocketMessage{
		Type: "addTask",
		Json: string(jsonDataTask),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg, writeLock)
		if err != nil {
			return err
		}
		return err
	}

	// Set the task and worker as not working
	err = database.SetTaskStatus(db, task.ID, "deleted", verbose, debug, wg)
	if err != nil {
		return err
	}
	err = database.SubtractWorkerIddleThreads1(db, worker.Name, verbose, debug, wg)
	if err != nil {
		return err
	}

	if verbose {
		log.Println("Delete Task send successfully")
	}
	/*
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
			if debug {
				log.Println("POST request was successful")
			}
			// Set the task and worker as not working
			err := database.SetTaskStatus(db, task.ID, "deleted", verbose, debug, wg)
			if err != nil {
				return err
			}
			err = database.SubtractWorkerIddleThreads1(db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}
		} else {
			message := "POST request failed with status:" + resp.Status + ". worker problably working"
			return fmt.Errorf(message)
		}
	*/
	return nil
}

/*
// SendGetTask sends a request to a worker to get the status of a task.
func SendGetTask(db *sql.DB, OauthTokenWorkers string, worker *globalstructs.Worker, task globalstructs.Task, verbose, debug bool) (globalstructs.Task, error) {
	return task, nil
}
*/

// CreateTLSClientWithCACert from cert.pem
func CreateTLSClientWithCACert(caCertPath string, verifyAltName, verbose, debug bool) (*http.Client, error) {
	// Load CA certificate from file
	caCert, err := os.ReadFile(caCertPath)
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

func WorkerDisconnected(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	if debug {
		log.Println("Error: WriteControl cant connect", worker.Name)
	}
	delete(config.WebSockets, worker.Name)

	err := database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
	if err != nil {
		return err
	}

	// Set as 'pending' all workers tasks to REDO
	err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
	if err != nil {
		return err
	}

	return nil
}
