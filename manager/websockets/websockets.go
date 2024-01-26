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
	var worker globalstructs.Worker
	for {
		response := globalstructs.WebsocketMessage{
			Type: "",
			JSON: "",
		}

		_, p, err := conn.ReadMessage()
		if err != nil {
			// if the clients conexion is down, this is the first error
			if debug {
				log.Println("WebSockets client conexion down error: ", err)
			}
			// check if worker not init
			if worker != (globalstructs.Worker{}) {
				err = utils.WorkerDisconnected(db, config, &worker, verbose, debug, wg)
				if err != nil {
					if debug {
						log.Println("WebSockets WorkerDisconnected error: ", err)
					}
				}
			} else {
				if debug {
					log.Println("WebSockets Worker empty")
				}
			}
			return
		}

		var msg globalstructs.WebsocketMessage
		err = json.Unmarshal(p, &msg)
		if err != nil {
			if debug {
				log.Println("WebSockets Error decoding JSON:", err)
			}
			continue
		}

		switch msg.Type {
		case "addWorker":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}

			err = json.Unmarshal([]byte(msg.JSON), &worker)
			if err != nil {
				log.Println("WebSockets addWorker Unmarshal error: ", err)
			}
			// add con to worker
			err = addWorker(worker, db, verbose, debug, wg)
			if err != nil {
				log.Println("WebSockets addWorker error: ", err)
				response.Type = "FAILED"
			} else {
				response.Type = "OK"
				config.WebSockets[worker.Name] = conn
			}

		case "deleteWorker":
			if debug {
				log.Println(msg.Type)
			}
			err = json.Unmarshal([]byte(msg.JSON), &worker)
			if err != nil {
				log.Println("WebSockets deleteWorker Unmarshal error: ", err)
			}

			err = database.RmWorkerName(db, worker.Name, verbose, debug, wg)
			if err != nil {
				log.Println("WebSockets RmWorkerName error: ", err)
				response.Type = "FAILED"
			} else {
				response.Type = "OK"
				config.WebSockets[worker.Name].Close()
				delete(config.WebSockets, worker.Name)
			}

			// Set the tasks as failed
			err := database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
			if err != nil {
				log.Println("WebSockets SetTasksWorkerFailed error: ", err)
			}
		case "callbackTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}

			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.JSON), &result)
			if err != nil {
				log.Println("WebSockets addWorker Unmarshal error: ", err)
			}

			err = callback(result, config, db, verbose, debug, wg)

			if err != nil {
				log.Println("WebSockets callbackTask error: ", err)
			}

			//Responses

		case "OK;addTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}
			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.JSON), &result)
			if err != nil {
				log.Println("WebSockets addWorker Unmarshal error: ", err)
			}

			// Set task as executed
			err = database.SetTaskExecutedAtNow(db, result.ID, verbose, debug, wg)
			if err != nil {
				log.Println("WebSockets Error SetTaskExecutedAt in request:", err)
			}

			// Set workerName in DB and in object
			err = database.SetTaskWorkerName(db, result.ID, result.WorkerName, verbose, debug, wg)
			if err != nil {
				log.Println("WebSockets Error SetWorkerNameTask in request:", err)
			}

			if verbose {
				log.Println("WebSockets Task send successfully")
			}
		case "FAILED;addTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}

			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.JSON), &result)
			if err != nil {
				log.Println("WebSockets addWorker Unmarshal error: ", err)
			}

			// Set the task as pending because the worker return error in add, so its not been procesed
			err = database.SetTaskStatus(db, result.ID, "pending", verbose, debug, wg)
			if err != nil {
				if verbose {
					log.Println("WebSockets HandleCallback { \"error\" : \"Error SetTaskStatus: " + err.Error() + "\"}")
				}
				log.Println("WebSockets Error SetTaskStatus in request:", err)
			}
		case "OK;deleteTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}
		case "FAILED;deleteTask":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}
			log.Println("WebSockets ------------------ TODO FAILED;deleteTask")
		case "status":
			if debug {
				log.Println("WebSockets msg.Type", msg.Type)
				log.Println("WebSockets msg.JSON", msg.JSON)
			}
			if msg.Type == "status" {
				// Unmarshal the JSON into a WorkerStatus struct
				var status globalstructs.WorkerStatus
				err = json.Unmarshal([]byte(msg.JSON), &status)
				if err != nil {
					log.Println("WebSockets status Unmarshal error: ", err)
				}

				if verbose {
					log.Println("WebSockets Response status from worker", status.Name, msg.JSON)
				}
				worker, err := database.GetWorker(db, status.Name, verbose, debug)
				if err != nil {
					log.Println("WebSockets GetWorker error: ", err)
				}
				// If there is no error in making the request, assume worker is online
				err = database.SetWorkerUPto(true, db, &worker, verbose, debug, wg)
				if err != nil {
					log.Println("WebSockets status error: ", err)
				}

				// If worker IddleThreads is not the same as stored in the DB, update the DB
				if status.IddleThreads != worker.IddleThreads {
					err := database.SetIddleThreadsTo(status.IddleThreads, db, worker.Name, verbose, debug, wg)
					if err != nil {
						log.Println("WebSockets status SetIddleThreadsTo error: ", err)
					}
				}
			}
		}

		if debug {
			fmt.Printf("Received message type: %s\n", msg.Type)
			fmt.Printf("Received message json: %s\n", msg.JSON)
		}

		if response.Type != "" {
			jsonData, err := json.Marshal(response)
			if err != nil {
				log.Println("WebSockets Marshal error: ", err)
			}
			err = utils.SendMessage(conn, jsonData, verbose, debug, writeLock)
			if err != nil {
				log.Println("WebSockets SendMessage error: ", err)
			}
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
				err = database.SetWorkerUPto(true, db, &worker, verbose, debug, wg)
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

	// Handle the result as needed

	//Add 1 to Iddle thread in worker
	// add 1 when finish
	err = database.AddWorkerIddleThreads1(db, result.WorkerName, verbose, debug, wg)
	if err != nil {
		return err
	}

	return nil
}
