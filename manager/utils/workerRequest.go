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

func SendMessage(conn *websocket.Conn, message []byte, verbose, debug bool, writeLock *sync.Mutex) error {
	writeLock.Lock()
	defer writeLock.Unlock()
	if debug {
		log.Println("Utils SendMessage", string(message))
	}

	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}
	if debug {
		log.Println("Utils SendMessage OK", string(message))
	}
	return nil
}

// VerifyWorkersLoop checks and sets if the workers are UP infinitely.
func VerifyWorkersLoop(db *sql.DB, config *ManagerConfig, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	for {
		go verifyWorkers(db, config, verbose, debug, wg, writeLock)
		time.Sleep(time.Duration(config.StatusCheckSeconds) * time.Second)
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

// verifyWorker checks and sets if the worker is UP.
func verifyWorker(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	if debug {
		log.Println("Utils verifyWorker", worker.Name)
	}
	conn := config.WebSockets[worker.Name]
	if conn == nil {
		if debug {
			log.Println("Utils Error: The worker doesnt have a websocket", worker.Name)
		}

		delete(config.WebSockets, worker.Name)

		err := database.SetWorkerUPto(false, db, worker, verbose, debug, wg)
		if err != nil {
			return err
		}

		downCount, err := database.GetWorkerDownCount(db, worker, verbose, debug)
		if err != nil {
			return err
		}

		if downCount >= config.StatusCheckDown {
			if debug {
				log.Println("Utils downCount", downCount, " >= config.StatusCheckDown", config.StatusCheckDown)
			}
			// Set as 'pending' all workers tasks to REDO
			err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}

			err = database.RmWorkerName(db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}
		} else {
			err = database.AddWorkerDownCount(db, worker, verbose, debug, wg)
			if err != nil {
				return err
			}
		}

		return nil
	}

	msg := globalstructs.WebsocketMessage{
		Type: "status",
		JSON: "{}",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		if debug {
			log.Println("Utils Error: json.Marshal(msg):", err)
		}
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Utils Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg)
		if err != nil {
			return err
		}
		return err
	}

	// If no error worker is ok
	err = database.SetWorkerDownCount(0, db, worker, verbose, debug, wg)
	if err != nil {
		return err
	}

	return nil

}

// SendAddTask sends a request to a worker to add a task.
func SendAddTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	if debug {
		log.Println("Utils SendAddTask")
	}

	// Subtract Iddle thread in DB, in the next status to worker it will update to real data
	err := database.SubtractWorkerIddleThreads1(db, worker.Name, verbose, debug, wg)
	if err != nil {
		return err
	}

	conn := config.WebSockets[worker.Name]
	if conn == nil {
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
		JSON: string(jsonDataTask),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Utils Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg)
		if err != nil {
			return err
		}
		return err
	}

	// Set task as running
	err = database.SetTaskStatus(db, task.ID, "running", verbose, debug, wg)
	if err != nil {
		log.Println("Utils Error SetTaskStatus in request:", err)
	}

	// Set task as executed
	err = database.SetTaskExecutedAtNow(db, task.ID, verbose, debug, wg)
	if err != nil {
		return fmt.Errorf("Error SetTaskExecutedAt in request: %s", err)
	}

	// Set workerName in DB and in object
	err = database.SetTaskWorkerName(db, task.ID, worker.Name, verbose, debug, wg)
	if err != nil {
		return fmt.Errorf("Error SetWorkerNameTask in request: %s", err)
	}

	if verbose {
		log.Println("Utils Task send successfully")
	}

	return nil
}

// SendDeleteTask sends a request to a worker to stop and delete a task.
func SendDeleteTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) error {
	conn := config.WebSockets[worker.Name]
	if conn == nil {
		return fmt.Errorf("Error, websocket not found")
	}

	// Tast to json
	// Convert the struct to JSON
	jsonDataTask, err := json.Marshal(task)
	if err != nil {
		return err
	}

	msg := globalstructs.WebsocketMessage{
		Type: "deleteTask",
		JSON: string(jsonDataTask),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = SendMessage(conn, jsonData, verbose, debug, writeLock)
	if err != nil {
		if debug {
			log.Println("Utils Can't send message, error:", err)
		}
		err = WorkerDisconnected(db, config, worker, verbose, debug, wg)
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

	if verbose {
		log.Println("Utils Delete Task send successfully")
	}

	return nil
}

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
		log.Println("Utils verifyAltName YES", !verifyAltName)

		tlsConfig = &tls.Config{
			InsecureSkipVerify: false, // Ensure that server verification is enabled
			RootCAs:            certPool,
			MinVersion:         tls.VersionTLS12, // Set the desired minimum TLS version
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

func WorkerDisconnected(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	if debug {
		log.Println("Utils Error: WriteControl cant connect", worker.Name)
	}
	// Close connection
	if websocket, ok := config.WebSockets[worker.Name]; ok {
		websocket.Close()
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
