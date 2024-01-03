package websockets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

func GetWorkerMessage(conn *websocket.Conn, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	for {
		response := globalstructs.WebsocketMessage{
			Type: "",
			Json: "",
		}

		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var msg globalstructs.WebsocketMessage
		err = json.Unmarshal(p, &msg)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			continue
		}

		switch msg.Type {
		case "addWorker":
			if debug {
				log.Println("msg.Type", msg.Type)
				log.Println("msg.Json", msg.Json)
			}
			var worker globalstructs.Worker
			err = json.Unmarshal([]byte(msg.Json), &worker)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}
			// add con to worker
			err = addWorker(worker, db, verbose, debug, wg)
			if err != nil {
				log.Println("addWorker error: ", err)
				response.Type = "FAILED"
			} else {
				response.Type = "OK"
				config.WebSockets[worker.Name] = conn
			}

		case "deleteWorker":
			log.Println(msg.Type)
			var worker globalstructs.Worker
			err = json.Unmarshal([]byte(msg.Json), &worker)
			if err != nil {
				log.Println("deleteWorker Unmarshal error: ", err)
			}

			err = database.RmWorkerName(db, worker.Name, verbose, debug, wg)
			if err != nil {
				log.Println("RmWorkerName error: ", err)
				response.Type = "FAILED"
			} else {
				response.Type = "OK"
				delete(config.WebSockets, worker.Name)
			}
		case "callbackTask":
			if debug {
				log.Println("msg.Type", msg.Type)
				log.Println("msg.Json", msg.Json)
			}

			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.Json), &result)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}

			err = callback(result, config, db, verbose, debug, wg)

			if err != nil {
				log.Println("callbackTask error: ", err)
			}

			//Responses

		case "addTask":
			if debug {
				log.Println("msg.Type", msg.Type)
				log.Println("msg.Json", msg.Json)
			}
			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.Json), &result)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}

			// Set task as executed
			err = database.SetTaskExecutedAtNow(db, result.ID, verbose, debug, wg)
			if err != nil {
				log.Println("Error SetTaskExecutedAt in request:", err)
			}

			// Set workerName in DB and in object
			err = database.SetTaskWorkerName(db, result.ID, result.WorkerName, verbose, debug, wg)
			if err != nil {
				log.Println("Error SetWorkerNameTask in request:", err)
			}

			if verbose {
				log.Println("Task send successfully")
			}
		case "deleteTask":
			if debug {
				log.Println("msg.Type", msg.Type)
				log.Println("msg.Json", msg.Json)
			}
		case "status":
			if debug {
				log.Println("msg.Type", msg.Type)
				log.Println("msg.Json", msg.Json)
			}
			if msg.Type == "status" {
				// Unmarshal the JSON into a WorkerStatus struct
				var status globalstructs.WorkerStatus
				err = json.Unmarshal([]byte(msg.Json), &status)
				if err != nil {
					log.Println("status Unmarshal error: ", err)
				}

				log.Println("Response status from worker", status.Name, msg.Json)
				worker, err := database.GetWorker(db, status.Name, verbose, debug)
				// If there is no error in making the request, assume worker is online
				err = database.SetWorkerUPto(true, db, &worker, verbose, debug, wg)
				if err != nil {
					log.Println("status error: ", err)
				}

				// If worker status is not the same as stored in the DB, update the DB
				if status.IddleThreads != worker.IddleThreads {
					err := database.SetIddleThreadsTo(status.IddleThreads, db, worker.Name, verbose, debug, wg)
					if err != nil {
						log.Println("status SetIddleThreadsTo error: ", err)
					}
				}
			}
		}

		if debug {
			fmt.Printf("Received message type: %s\n", msg.Type)
			fmt.Printf("Received message json: %s\n", msg.Json)
		}

		if response.Type != "" {
			jsonData, err := json.Marshal(response)
			if err != nil {
				log.Println("Marshal error: ", err)
			}
			err = utils.SendMessage(conn, jsonData, verbose, debug, writeLock)
			if err != nil {
				log.Println("SendMessage error: ", err)
			}
		}
	}
}

func addWorker(worker globalstructs.Worker, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println("worker.Name", worker.Name)
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
				err = database.SetWorkerUPto(true, db, &worker, verbose, debug, wg)
				if err != nil {
					return err
				}

				// reset down count
				err = database.SetWorkerDownCount(0, db, &worker, verbose, debug, wg)
				if err != nil {
					return err
				}
				return err
			} else {
				return err
			}
		}

	}

	return nil
}

func callback(result globalstructs.Task, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println(result)
		log.Println("Received result (ID: ", result.ID, " from : ", result.WorkerName, " with command: ", result.Commands)
	}

	// Update task with the worker one
	err := database.UpdateTask(db, result, verbose, debug, wg)
	if err != nil {
		if verbose {
			log.Println("HandleCallback { \"error\" : \"Error UpdateTask: " + err.Error() + "\"}")
		}

		return err
	}

	// force set task done
	// Set the task as done
	err = database.SetTaskStatus(db, result.ID, "done", verbose, debug, wg)
	if err != nil {
		if verbose {
			log.Println("HandleCallback { \"error\" : \"Error SetTaskStatus: " + err.Error() + "\"}")
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

	// Handle the result as needed

	//Add 1 to Iddle thread in worker
	// add 1 when finish
	err = database.AddWorkerIddleThreads1(db, result.WorkerName, verbose, debug, wg)
	if err != nil {
		return err
	}

	return nil
}
