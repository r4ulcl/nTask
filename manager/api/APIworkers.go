package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
	"github.com/r4ulcl/nTask/manager/websockets"
)

// HandleWorkerGet Get handles the request to get workers
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
	username, ok := r.Context().Value(utils.UsernameKey).(string)
	if !ok {
		log.Println("API username", username)
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
		log.Println("API workers", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Use json.NewEncoder for safe encoding
	err = json.NewEncoder(w).Encode(workers)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid workers encode body:"+err.Error()+"\"}", http.StatusBadRequest)
	}
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
	_, okUser := r.Context().Value(utils.UsernameKey).(string)
	_, okWorker := r.Context().Value(utils.WorkerKey).(string)
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
		log.Println("API worker.Name", worker.Name)
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

// HandleWorkerPostWebsocket HandleWorkerPostWebsocket
func HandleWorkerPostWebsocket(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	_, okWorker := r.Context().Value(utils.WorkerKey).(string)
	if !okWorker {
		if verbose {
			log.Println("API HandleCallback: { \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	conn, err := globalstructs.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		if verbose {
			log.Println("API globalstructs.Upgrader.Upgrade connection down", err)
		}
		return
	}

	//go
	websockets.GetWorkerMessage(conn, config, db, verbose, debug, wg)

}

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
	_, okUser := r.Context().Value(utils.UsernameKey).(string)
	_, okWorker := r.Context().Value(utils.WorkerKey).(string)
	if !okUser && !okWorker {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	name := vars["NAME"]

	// TODO

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
	_, ok := r.Context().Value(utils.UsernameKey).(string)
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
		log.Println("API HandleWorkerStatus", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Use json.NewEncoder for safe encoding
	err = json.NewEncoder(w).Encode(worker)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid workers encode body:"+err.Error()+"\"}", http.StatusBadRequest)
	}
}

// Other functions

/*
// readUserIP reads the user's IP address from the request
func readUserIP(r *http.Request, verbose, debug bool) string {
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
*/
