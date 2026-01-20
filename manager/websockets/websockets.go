package websockets

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

// GetWorkerMessage processes worker messages with robust heartbeat and write synchronization.
func GetWorkerMessage(conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	var worker globalstructs.Worker
	// configure timing and retries
	const (
		pongWait        = 120 * time.Second
		pingInterval    = 30 * time.Second
		maxRecovery     = 5
		recoveryBackoff = pingInterval * 2
		writeTimeout    = 60 * time.Second
	)

	// protect writes
	var writeMu sync.Mutex
	// heartbeat tracking
	var lastPong time.Time = time.Now()
	var lastPongMu sync.Mutex

	// read limits and initial deadline
	conn.SetReadLimit(globalstructs.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(globalstructs.PongWait))
	conn.SetPongHandler(func(appData string) error {
		// update lastPong under lock
		lastPongMu.Lock()
		lastPong = time.Now()
		lastPongMu.Unlock()
		// extend read deadline
		conn.SetReadDeadline(time.Now().Add(globalstructs.PongWait))
		if debug {
			log.Println("Received Pong from worker", worker.Name)
		}
		return nil
	})

	// handle client Close frames
	conn.SetCloseHandler(func(code int, text string) error {
		if debug {
			log.Printf("Received Close frame (code=%d): %s", code, text)
		}
		// immediate shutdown
		return conn.Close()
	})

	// ping ticker + recovery
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			// send ping under write lock
			writeMu.Lock()
			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout))
			writeMu.Unlock()
			if err != nil {
				if debug {
					log.Println("Ping write failed, closing:", err)
				}
				conn.Close()
				return
			}

			// check time since lastPong
			lastPongMu.Lock()
			elapsed := time.Since(lastPong)
			lastPongMu.Unlock()
			if elapsed > globalstructs.PongWait {
				if debug {
					log.Println("Missed heartbeat—entering recovery retries")
				}
				// recovery loop
				for i := 1; i <= maxRecovery; i++ {
					time.Sleep(recoveryBackoff)
					if debug {
						log.Printf("Recovery ping #%d", i)
					}
					writeMu.Lock()
					conn.SetWriteDeadline(time.Now().Add(writeTimeout))
					err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout))
					writeMu.Unlock()
					if err != nil {
						if debug {
							log.Println("Recovery ping failed, closing:", err)
						}
						conn.Close()
						return
					}
					lastPongMu.Lock()
					elapsed = time.Since(lastPong)
					lastPongMu.Unlock()
					if elapsed <= globalstructs.PongWait {
						if debug {
							log.Println("Heartbeat recovered on attempt", i)
						}
						break
					}
				}
				// final check
				lastPongMu.Lock()
				elapsed = time.Since(lastPong)
				lastPongMu.Unlock()
				if elapsed > globalstructs.PongWait {
					if debug {
						log.Println("No heartbeat after recovery—disconnecting")
					}
					conn.Close()
					return
				}
			}
		}
	}()

	// main read loop
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			if debug {
				log.Println("ReadMessage error (disconnect):", err)
			}
			handleConnectionError(err, db, config, &worker, verbose, debug)
			return
		}

		msg, err := parseMessage(payload, debug)
		if err != nil {
			if debug {
				log.Println("parseMessage error:", err)
			}
			continue
		}
		handleMessage(msg, conn, config, db, &worker, verbose, debug)
	}
}

func handleConnectionError(
	err error,
	db *sql.DB,
	config *utils.ManagerConfig,
	worker *globalstructs.Worker,
	verbose, debug bool,
) {
	if debug {
		log.Println("WebSocket connection error:", err)
	}
	// only call WorkerDisconnected if worker has been initialized
	if *worker != (globalstructs.Worker{}) {
		if err := utils.WorkerDisconnected(db, config, worker, verbose, debug); err != nil && debug {
			log.Println("WorkerDisconnected error:", err)
		}
	} else if debug {
		log.Println("Worker is uninitialized; nothing to clean up")
	}
}

func parseMessage(p []byte, debug bool) (globalstructs.WebsocketMessage, error) {
	var msg globalstructs.WebsocketMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		if debug {
			log.Println("Error decoding JSON:", err)
		}
		return msg, err
	}
	return msg, nil
}

func handleMessage(msg globalstructs.WebsocketMessage, conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool) {
	switch msg.Type {
	case "addWorker":
		handleAddWorker(msg, conn, config, db, worker, verbose, debug)
	case "deleteWorker":
		handleDeleteWorker(msg, config, db, worker, verbose, debug)
	case "callbackTask":
		handleCallbackTask(msg, config, db, verbose, debug)
	case "status":
		handleWorkerStatus(msg, db, verbose, debug)
	case "OK;addTask":
		if debug {
			log.Println("Receive message OK;addTask from worker")
		}
		// Set here as working?
	case "OK;deleteTask":
		if debug {
			log.Println("Received OK;deleteTask — marking task as deleted")
		}
		/*var completedTask globalstructs.Task
		if err := json.Unmarshal([]byte(msg.JSON), &completedTask); err != nil {
			log.Println("Error unmarshaling OK;deleteTask JSON:", err)
			break
		}

		if err := database.SetTaskStatus(db, completedTask.ID, "deleted", verbose, debug, wg); err != nil {
			log.Println("Error setting task status to deleted:", err)
		}*/

	case "FAILED;deleteTask":
		if debug {
			log.Println("Received FAILED;deleteTask — deletion failed, leaving state or retrying")
		}
		var failedDelTask globalstructs.Task
		if err := json.Unmarshal([]byte(msg.JSON), &failedDelTask); err != nil {
			log.Println("Error unmarshaling FAILED;deleteTask JSON:", err)
			break
		}
		log.Printf("Task %d could not be killed on worker %q", failedDelTask.ID, failedDelTask.WorkerName)

	case "FAILED;addTask":
		if debug {
			log.Println("Receive message FAILED;addTask from worker - re‐queueing", msg.JSON)
		}
		var failedTask globalstructs.Task
		if err := json.Unmarshal([]byte(msg.JSON), &failedTask); err != nil {
			log.Println("Error unmarshaling FAILED;addTask JSON:", err)
			break
		}

		// Revert the task to “pending” so it can be retried
		// Wait to avoid updating the time at the same second
		time.Sleep(1 * time.Second)

		if err := database.SetTaskStatus(db, failedTask.ID, "pending", verbose, debug); err != nil {
			log.Println("Error setting task status back to pending:", err)
		}
	default:
		if debug {
			log.Printf("--------- Unhandled message type: %s\n", msg.Type)
		}
	}
}

func handleAddWorker(msg globalstructs.WebsocketMessage, conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool) {
	if err := handleWorkerMessage(msg, worker, db, verbose, debug, func() error {
		config.WebSockets[worker.Name] = conn
		return addWorker(*worker, db, verbose, debug)
	}); err != nil {
		log.Println("Error handling addWorker:", err)
	}
}

func handleDeleteWorker(msg globalstructs.WebsocketMessage, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool) {
	if err := handleWorkerMessage(msg, worker, db, verbose, debug, func() error {
		return database.RmWorkerName(db, worker.Name, verbose, debug)
	}); err != nil {
		log.Println("Error handling deleteWorker:", err)
	}
}

func handleWorkerMessage(msg globalstructs.WebsocketMessage, worker *globalstructs.Worker, db *sql.DB, verbose, debug bool, workerAction func() error) error {
	if debug {
		log.Println("Handling worker message")
	}
	if err := json.Unmarshal([]byte(msg.JSON), worker); err != nil {
		log.Println("Error unmarshaling worker message:", err)
		return err
	}

	if err := workerAction(); err != nil {
		return err
	}

	return nil
}

func handleCallbackTask(msg globalstructs.WebsocketMessage, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	if debug {
		log.Println("Handling callbackTask message")
	}
	var task globalstructs.Task
	if err := json.Unmarshal([]byte(msg.JSON), &task); err != nil {
		log.Println("Error unmarshaling callbackTask message:", err)
		return
	}
	if err := callback(task, config, db, verbose, debug); err != nil {
		log.Println("Error handling callback task:", err)
	}
}

func handleWorkerStatus(msg globalstructs.WebsocketMessage, db *sql.DB, verbose, debug bool) {
	if debug {
		log.Println("Handling status message")
	}
	var status globalstructs.WorkerStatus
	if err := json.Unmarshal([]byte(msg.JSON), &status); err != nil {
		log.Println("Error unmarshaling status message:", err)
		return
	}
	worker, err := database.GetWorker(db, status.Name, verbose, debug)
	if err != nil {
		log.Println("Error retrieving worker from database:", err)
		return
	}
	if err := database.SetWorkerUPto(db, worker.Name, true, verbose, debug); err != nil {
		log.Println("Error setting worker status to UP:", err)
	}

	if err := database.SetWorkerDownCount(db, worker.Name, 0, verbose, debug); err != nil {
		log.Println("Error setting worker status to UP:", err)
	}
	if status.IddleThreads != worker.IddleThreads {
		if err := database.SetIddleThreadsTo(db, worker.Name, status.IddleThreads, verbose, debug); err != nil {
			log.Println("Error updating idle threads in database:", err)
		}
	}
}

func addWorker(worker globalstructs.Worker, db *sql.DB, verbose, debug bool) error {

	if debug {
		log.Println("WebSockets worker.Name", worker.Name)
	}

	err := database.AddWorker(db, &worker, verbose, debug)
	if err != nil {
		err = utils.HandleAddWorkerError(err, db, &worker, verbose, debug)
		if err != nil {
			return err
		}
	}

	return nil
}

func callback(result globalstructs.Task, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) error {

	if debug {
		log.Println("WebSockets result: ", result)
		log.Println("WebSockets Received result (ID: ", result.ID, " from : ", result.WorkerName, " with command: ", result.Commands)
	}

	// Update task with the worker one
	err := database.UpdateTask(db, result, verbose, debug)
	if err != nil {
		if debug || verbose {
			log.Println("WebSockets HandleCallback { \"error\" : \"Error UpdateTask: " + err.Error() + "\"}")
		}

		return err
	}

	// if callbackURL is not empty send the request to the client
	if result.CallbackURL != "" {
		utils.CallbackUserTaskMessage(config, &result, verbose, debug)
	}

	// if path not empty
	if config.DiskPath != "" {
		//get the task from DB to get updated
		task, err := database.GetTask(db, result.ID, verbose, debug)
		if err != nil {
			return err
		}
		err = utils.SaveTaskToDisk(task, config.DiskPath, verbose, debug)
		if err != nil {
			return err
		}
	}

	return nil
}
