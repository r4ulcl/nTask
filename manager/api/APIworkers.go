package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

// HandleWorkerGet handles the request to get workers
// @description Handle worker request
// @summary Get workers
// @Tags worker
// @accept application/json
// @produce application/json
// @success 200 {array} globalstructs.Worker
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /worker [get]
func HandleWorkerGet(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	username, ok := r.Context().Value("username").(string)
	if !ok {
		log.Println(username)
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	// get workers
	workers, err := database.GetWorkers(db, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	jsonData, err := json.Marshal(workers)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Marshal body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	if debug {
		// Print the JSON data
		log.Println(string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// HandleWorkerPost handles the request to add a worker
// @description Add a worker, normally done by the worker
// @summary Add a worker
// @Tags worker
// @accept application/json
// @produce application/json
// @param worker body globalstructs.Worker true "Worker object to create"
// @success 200 {array} globalstructs.Worker
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /worker [post]
func HandleWorkerPost(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	_, okUser := r.Context().Value("username").(string)
	_, okWorker := r.Context().Value("worker").(string)
	if !okUser && !okWorker {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	var worker globalstructs.Worker

	err := json.NewDecoder(r.Body).Decode(&worker)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Decode body: "+err.Error()+"\"}", http.StatusBadRequest)
	}

	IP := ReadUserIP(r, verbose, debug)

	err = addWorker(worker, IP, db, verbose, debug, wg)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Decode body: "+err.Error()+"\"}", http.StatusBadRequest)
	}

	// Handle the result as needed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func addWorker(worker globalstructs.Worker, ip string, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println("worker.Name", worker.Name, "worker.IP", worker.IP, "worker.Name", worker.Name)
	}

	worker.IP = ip

	err := database.AddWorker(db, &worker, verbose, debug, wg)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 { // MySQL error number for duplicate entry
				// Set as 'pending' all workers tasks to REDO
				err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
				if err != nil {
					return err
				}

				//Update oauth key
				err := database.SetWorkerOauthToken(worker.OauthToken, db, &worker, verbose, debug, wg)
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

func HandleWorkerPostWebsocket(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	_, okWorker := r.Context().Value("worker").(string)
	if !okWorker {
		if verbose {
			log.Println("HandleCallback: { \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	conn, err := globalstructs.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	//go func() {
	for {
		response := globalstructs.WebsocketMessage{
			Type: "",
			Json: "",
		}

		messageType, p, err := conn.ReadMessage()
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
			log.Println(msg.Type)
			var worker globalstructs.Worker
			err = json.Unmarshal([]byte(msg.Json), &worker)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}
			// add con to worker
			err = addWorker(worker, "127.0.0.127", db, verbose, debug, wg)
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
			log.Println(msg.Type)

			var result globalstructs.Task
			err = json.Unmarshal([]byte(msg.Json), &result)
			if err != nil {
				log.Println("addWorker Unmarshal error: ", err)
			}

			err = callback(result, config, db, verbose, debug, wg)

			if err != nil {
				log.Println("callbackTask error: ", err)
				response.Type = "FAILED"
			} else {
				response.Type = "OK"
			}
		case "addTask":
			log.Println(msg.Type)

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
			log.Println(msg.Type)
		case "status":
			log.Println(msg.Type)
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
			err = conn.WriteMessage(messageType, jsonData)
			if err != nil {
				log.Println("error WriteMessage", err)
			}
		}
	}
	//}()

}

/*
func (m *WebSocketManager) broadcastMessage(msg []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for conn := range m.connections {
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error broadcasting message:", err)
		}
	}
}*/

// HandleWorkerDeleteName handles the request to remove a worker
// @description Remove a worker from the system
// @summary Remove a worker
// @Tags worker
// @accept application/json
// @produce application/json
// @param NAME path string true "Worker NAME"
// @success 200 {array} string
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /worker/{NAME} [delete]
func HandleWorkerDeleteName(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	_, okUser := r.Context().Value("username").(string)
	_, okWorker := r.Context().Value("worker").(string)
	if !okUser && !okWorker {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	name := vars["NAME"]

	err := database.RmWorkerName(db, name, verbose, debug, wg)
	if err != nil {
		http.Error(w, "{ \"error\" : \"RmWorkerName: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "{\"status\": \"OK\"}")
}

// HandleWorkerStatus returns the status of a worker
// @description Get status of worker
// @summary Get status of worker
// @Tags worker
// @accept application/json
// @produce application/json
// @param NAME path string true "Worker NAME"
// @success 200 {object} globalstructs.Worker
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /worker/{NAME} [get]
func HandleWorkerStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	name := vars["NAME"]

	worker, err := database.GetWorker(db, name, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid GetWorker body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	jsonData, err := json.Marshal(worker)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Marshal body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	if debug {
		// Print the JSON data
		log.Println("HandleWorkerStatus", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// Other functions

// ReadUserIP reads the user's IP address from the request
func ReadUserIP(r *http.Request, verbose, debug bool) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	// Split IP address and port
	ip, _, err := net.SplitHostPort(IPAddress)
	if err == nil {
		return ip
	}

	// If there's an error (e.g., no port found), return the original address
	return IPAddress
}
