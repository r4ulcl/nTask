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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
)

// SendMessage Function to send message in a websocket
func SendMessage(conn *websocket.Conn, message []byte, verbose, debug bool, writeLock *sync.Mutex) error {
	writeLock.Lock()
	defer writeLock.Unlock()
	if debug {
		log.Println("Utils SendMessage", string(message))
	}
	writeTimeout := 10 * time.Second
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
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
func VerifyWorkersLoop(db *sql.DB, config *ManagerConfig, verbose, debug bool, writeLock *sync.Mutex) {
	ticker := time.NewTicker(time.Duration(config.StatusCheckSeconds) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		verifyWorkers(db, config, verbose, debug, writeLock)
	}
}

// DeleteMaxTaskHistoryLoop Loop and Delete Database Entries if num tasks > config.MaxTaskHistory
func DeleteMaxTaskHistoryLoop(db *sql.DB, config *ManagerConfig, verbose, debug bool) {
	maxEntries := config.MaxTaskHistory
	tableName := "task"
	if maxEntries > 0 {
		for {
			err := database.DeleteMaxEntriesHistory(db, maxEntries, tableName, verbose, debug)
			if err != nil && (verbose || debug) {
				log.Println("Error DeleteMaxEntriesHistory:", err)
			}
			time.Sleep(1 * time.Hour)
		}
	}
}

// getWorkersThreads get DefaultThreads of all workers
func getWorkersThreads(db *sql.DB, verbose, debug bool) int {

	workersThreads := 0
	// Get all workers from the database
	workers, err := database.GetWorkerUP(db, verbose, debug)
	if err != nil {
		log.Print("GetWorker", err)
	}

	// Verify each worker
	for _, worker := range workers {
		workersThreads += worker.DefaultThreads
	}

	if debug {
		log.Println("getWorkersThreads workersThreads", workersThreads)
	}

	return workersThreads
}

// verifyWorkers checks and sets if the workers are UP.
func verifyWorkers(db *sql.DB, config *ManagerConfig, verbose, debug bool, writeLock *sync.Mutex) {
	// Get all workers from the database
	workers, err := database.GetWorkers(db, verbose, debug)
	if err != nil {
		log.Print("GetWorker", err)
	}

	// Verify each worker
	for _, worker := range workers {
		err := verifyWorker(db, config, &worker, verbose, debug, writeLock)
		if err != nil {
			log.Print("verifyWorker ", err)
		}
	}
}

// verifyWorker checks and sets if the worker is UP.
func verifyWorker(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, writeLock *sync.Mutex) error {
	if debug {
		log.Println("Utils verifyWorker", worker.Name)
	}

	conn := config.WebSockets[worker.Name]
	if conn == nil {
		return handleMissingWebSocket(worker, db, config, verbose, debug)
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
	return SendMessage(conn, jsonData, verbose, debug, writeLock)
}

// handleMissingWebSocket marks a worker down (or removes it) when its WS is gone.
// It tolerates “not found” from SetWorkerUPto in case the row was already deleted.
func handleMissingWebSocket(
	worker *globalstructs.Worker,
	db *sql.DB,
	config *ManagerConfig,
	verbose, debug bool,
) error {
	if debug {
		log.Println("Utils Error: no websocket for worker", worker.Name)
	}

	// Remove from in-memory map
	delete(config.WebSockets, worker.Name)

	// Mark as down
	if err := database.SetWorkerUPto(db, worker.Name, false, verbose, debug); err != nil {
		// ignore “not found” since it may have been removed already
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		if debug {
			log.Printf("Utils handleMissingWebSocket: %v", err)
		}
	}

	// Increment down-count
	downCount, err := database.GetWorkerDownCount(db, worker.Name, verbose, debug)
	if err != nil {
		return err
	}
	if err := database.AddWorkerDownCount(db, worker.Name, verbose, debug); err != nil {
		return err
	}

	// If exceeded retries, orphan tasks and delete the worker
	if downCount+1 >= config.StatusCheckDown {
		if err := database.SetTasksWorkerPending(db, worker.Name, verbose, debug); err != nil {
			return err
		}
		if err := database.RmWorkerName(db, worker.Name, verbose, debug); err != nil {
			return err
		}
	}

	return nil
}

// sendAddTask sends a request to a worker to add a task.
func sendAddTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
	if debug {
		log.Println("Utils sendAddTask")
	}

	// Subtract Iddle thread in DB, in the next status to worker it will update to real data
	err := database.SubtractWorkerIddleThreads1(db, worker.Name, verbose, debug)
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
		return err
	}

	task.Status = "running"
	task.WorkerName = worker.Name

	// Set task as running
	err = database.UpdateTask(db, *task, verbose, debug)
	if err != nil {
		return fmt.Errorf("Utils Error SetTaskStatus in request: %s", err)
	}

	// Set task as executed
	err = database.SetTaskExecutedAtNow(db, task.ID, verbose, debug)
	if err != nil {
		return fmt.Errorf("Error SetTaskExecutedAt in request: %s", err)
	}

	if verbose {
		log.Println("Utils Task send successfully")
	}

	return nil
}

// SendDeleteTask sends a request to a worker to stop and delete a task.
func SendDeleteTask(db *sql.DB, config *ManagerConfig, worker *globalstructs.Worker, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) error {
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
		return err
	}

	// Set the task and worker as not working
	err = database.SetTaskStatus(db, task.ID, "deleted", verbose, debug)
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
			MinVersion:         tls.VersionTLS12, // Minimum version set to TLS 1.2
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

// WorkerDisconnected is called when a live connection error occurs.
// It closes the socket, marks the worker down, and re-queues its tasks.
// It tolerates “not found” from SetWorkerUPto.
func WorkerDisconnected(
	db *sql.DB,
	config *ManagerConfig,
	worker *globalstructs.Worker,
	verbose, debug bool,
) error {
	if debug {
		log.Println("Utils WorkerDisconnected: closing websocket for", worker.Name)
	}

	// Close the socket if still present
	if ws, ok := config.WebSockets[worker.Name]; ok {
		ws.Close()
	}
	delete(config.WebSockets, worker.Name)

	// Mark as down
	if err := database.SetWorkerUPto(db, worker.Name, false, verbose, debug); err != nil {
		// ignore “not found” since it may have been removed already
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		if debug {
			log.Printf("Utils WorkerDisconnected: %v", err)
		}
	}

	// Re-queue any in-flight tasks
	if err := database.SetTasksWorkerPending(db, worker.Name, verbose, debug); err != nil {
		return err
	}

	return nil
}
