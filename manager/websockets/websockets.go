package websockets

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

// GetWorkerMessage Process worker message, add, delete, status, etc
func GetWorkerMessage(conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	var worker globalstructs.Worker

	setPongHandler(conn, &worker, debug)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sendPing(conn, &worker, debug)
		default:
			processIncomingMessage(conn, config, db, &worker, verbose, debug, wg)
		}
	}
}

func setPongHandler(conn *websocket.Conn, worker *globalstructs.Worker, debug bool) {
	conn.SetPongHandler(func(appData string) error {
		if debug {
			log.Println("Received Pong:", appData, worker.Name)
		}
		return nil
	})
}

func sendPing(conn *websocket.Conn, worker *globalstructs.Worker, debug bool) {
	if debug {
		log.Println("Send Ping:", worker.Name)
	}
	if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(2*time.Second)); err != nil {
		log.Println("Error sending Ping", worker.Name, ":", err)
	}
}

func processIncomingMessage(conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) {
	_, p, err := conn.ReadMessage()
	if err != nil {
		handleConnectionError(err, db, config, worker, verbose, debug, wg)
		return
	}

	msg, err := parseMessage(p, debug)
	if err != nil {
		return
	}

	handleMessage(msg, conn, config, db, worker, verbose, debug, wg)
}

func handleConnectionError(err error, db *sql.DB, config *utils.ManagerConfig, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) {
	if debug {
		log.Println("WebSocket connection error:", err)
	}
	if *worker != (globalstructs.Worker{}) {
		if err := utils.WorkerDisconnected(db, config, worker, verbose, debug, wg); err != nil && debug {
			log.Println("WorkerDisconnected error:", err)
		}
	} else if debug {
		log.Println("Worker is uninitialized")
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

func handleMessage(msg globalstructs.WebsocketMessage, conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) {
	switch msg.Type {
	case "addWorker":
		handleAddWorker(msg, conn, config, db, worker, verbose, debug, wg)
	case "deleteWorker":
		handleDeleteWorker(msg, config, db, worker, verbose, debug, wg)
	case "callbackTask":
		handleCallbackTask(msg, config, db, verbose, debug, wg)
	case "status":
		handleWorkerStatus(msg, db, verbose, debug, wg)
	default:
		if debug {
			log.Printf("Unhandled message type: %s\n", msg.Type)
		}
	}
}

func handleAddWorker(msg globalstructs.WebsocketMessage, conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) {
	if debug {
		log.Println("Handling addWorker message")
	}
	if err := json.Unmarshal([]byte(msg.JSON), worker); err != nil {
		log.Println("Error unmarshaling addWorker message:", err)
		return
	}
	if err := addWorker(*worker, db, verbose, debug, wg); err != nil {
		log.Println("Error adding worker:", err)
	} else {
		config.WebSockets[worker.Name] = conn
	}
}

func handleDeleteWorker(msg globalstructs.WebsocketMessage, config *utils.ManagerConfig, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) {
	if debug {
		log.Println("Handling deleteWorker message")
	}
	if err := json.Unmarshal([]byte(msg.JSON), worker); err != nil {
		log.Println("Error unmarshaling deleteWorker message:", err)
		return
	}
	if err := database.RmWorkerName(db, worker.Name, verbose, debug, wg); err != nil {
		log.Println("Error removing worker:", err)
	} else {
		delete(config.WebSockets, worker.Name)
	}
}

func handleCallbackTask(msg globalstructs.WebsocketMessage, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	if debug {
		log.Println("Handling callbackTask message")
	}
	var task globalstructs.Task
	if err := json.Unmarshal([]byte(msg.JSON), &task); err != nil {
		log.Println("Error unmarshaling callbackTask message:", err)
		return
	}
	if err := callback(task, config, db, verbose, debug, wg); err != nil {
		log.Println("Error handling callback task:", err)
	}
}

func handleWorkerStatus(msg globalstructs.WebsocketMessage, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
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
	if err := database.SetWorkerUPto(true, db, &worker, verbose, debug, wg); err != nil {
		log.Println("Error setting worker status to UP:", err)
	}
	if status.IddleThreads != worker.IddleThreads {
		if err := database.SetIddleThreadsTo(status.IddleThreads, db, worker.Name, verbose, debug, wg); err != nil {
			log.Println("Error updating idle threads in database:", err)
		}
	}
}

func addWorker(worker globalstructs.Worker, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println("WebSockets worker.Name", worker.Name)
	}

	err := database.AddWorker(db, &worker, verbose, debug, wg)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 { // MySQL error number for duplicate entry
				// Set as 'pending' all workers tasks to REDO
				err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
				if err != nil {
					return err
				}

				// set worker up
				err = database.UpdateWorker(db, &worker, verbose, debug, wg)
				if err != nil {
					return err
				}

				// reset down count
				err = database.SetWorkerDownCount(0, db, &worker, verbose, debug, wg)
				if err != nil {
					return err
				}
			}
			return err

		}

	}

	return nil
}

func callback(result globalstructs.Task, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println("WebSockets result: ", result)
		log.Println("WebSockets Received result (ID: ", result.ID, " from : ", result.WorkerName, " with command: ", result.Commands)
	}

	// Update task with the worker one
	err := database.UpdateTask(db, result, verbose, debug, wg)
	if err != nil {
		if verbose {
			log.Println("WebSockets HandleCallback { \"error\" : \"Error UpdateTask: " + err.Error() + "\"}")
		}

		return err
	}

	// force set task to status receive
	// Set the task as done
	if result.Status == "failed" {
		err = database.SetTaskStatus(db, result.ID, result.Status, verbose, debug, wg)
		if err != nil {
			if verbose {
				log.Println("WebSockets HandleCallback { \"error\" : \"Error SetTaskStatus: " + err.Error() + "\"}")
			}
			return err
		}
	} else {
		err = database.SetTaskStatus(db, result.ID, "done", verbose, debug, wg)
		if err != nil {
			if verbose {
				log.Println("WebSockets HandleCallback { \"error\" : \"Error SetTaskStatus: " + err.Error() + "\"}")
			}
			return err
		}
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
