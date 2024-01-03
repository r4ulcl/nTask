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
	"github.com/r4ulcl/nTask/manager/websockets"
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

	err = addWorker(worker, db, verbose, debug, wg)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Decode body: "+err.Error()+"\"}", http.StatusBadRequest)
	}

	// Handle the result as needed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

func HandleWorkerPostWebsocket(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
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

	//go
	websockets.GetWorkerMessage(conn, config, db, verbose, debug, wg, writeLock)

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
