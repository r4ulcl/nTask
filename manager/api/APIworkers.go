package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

// HandleCallback handles the callback from a slave
// @Summary Handle callback from slave
// @Description Handle callback from slave
// @Tags callback
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 "OK"
// @Failure 400 {string} string "Invalid callback body"
// @Failure 401 {string} string "Unauthorized"
// @Router /callback [post]
func HandleCallback(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken, verbose) && incorrectOauthWorker(oauthKey, config.OauthTokenWorkers, verbose) {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}
	var result globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+"", http.StatusBadRequest)
		return
	}

	if verbose {
		log.Println(result)
		log.Println("Received result (ID: ", result.ID, " from : ", result.WorkerName, " with command: ", result.Commands)
	}

	// Update task with the worker one
	err = database.UpdateTask(db, result, verbose)
	if err != nil {
		http.Error(w, "Error UpdateTask ", http.StatusBadRequest)
		return
	}

	//Add value to thread working
	err = database.AddWorkerIddleThreads1(db, result.WorkerName, verbose)
	if err != nil {
		http.Error(w, "Error AddWorkerIddleThreads1 ", http.StatusBadRequest)
		return
	}

	// if callbackURL is not empty send the request to the client
	if config.CallbackURL != "" {
		utils.CallbackUserTaskMessage(config, &result, verbose)
	}

	// if path not empty
	if config.DiskPath != "" {
		//get the task from DB to get updated
		task, err := database.GetTask(db, result.ID, verbose)
		if err != nil {
			log.Println(err)
		}
		err = utils.SaveTaskToDisk(task, config.DiskPath, verbose)
		if err != nil {
			log.Println(err)
		}
	}

	// Handle the result as needed

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// HandleWorkerGet handles the request to get workers
// @Summary Get workers
// @Description Handle worker request
// @Tags worker
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {string} string "OK"
// @Router /worker [get]
func HandleWorkerGet(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken, verbose) {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	// get workers
	workers, err := database.GetWorkers(db, verbose)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+"", http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(workers)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+"", http.StatusBadRequest)
		return
	}

	if verbose {
		// Print the JSON data
		log.Println(string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// HandleWorkerPost handles the request to add a worker
// @Summary Add a worker
// @Description Add a worker, normally done by the worker
// @Tags worker
// @Accept json
// @Produce json
// @Success 200 {string} string
// @Router /worker [post]
// @Type basic
// @In header
// @Name Authorization
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Param worker body globalstructs.Worker true "Worker object to create"
func HandleWorkerPost(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken, verbose) && incorrectOauthWorker(oauthKey, config.OauthTokenWorkers, verbose) {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	var request globalstructs.Worker
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+"", http.StatusBadRequest)
		return
	}

	request.IP = ReadUserIP(r, verbose)

	if verbose {
		log.Println("request.Name", request.Name, "request.IP", request.IP, "request.Name", request.Name)
	}

	err = database.AddWorker(db, &request, verbose)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 { // MySQL error number for duplicate entry
				// Set as 'failed' all workers tasks
				err = database.SetTasksWorkerFailed(db, request.Name, verbose)
				if err != nil {
					return
				}

				//Update oauth key
				err := database.SetWorkerOauthToken(request.OauthToken, db, &request, verbose)
				if err != nil {
					http.Error(w, "Error setWorkerUp ", http.StatusBadRequest)
					return
				}

				// set worker up
				err = database.SetWorkerUPto(true, db, &request, verbose)
				if err != nil {
					http.Error(w, "Error setWorkerUp ", http.StatusBadRequest)
					return
				}

				// reset down count
				err = database.SetWorkerDownCount(0, db, &request, verbose)
				if err != nil {
					http.Error(w, "Error SetWorkerCount ", http.StatusBadRequest)
					return
				}
				return
			} else {
				message := "Invalid worker info: " + err.Error()
				http.Error(w, message, http.StatusBadRequest)
				return
			}
		}

	}

	// Handle the result as needed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Worker with Name %s added", request.Name)
}

// HandleWorkerDeleteName handles the request to remove a worker
// @Summary Remove a worker
// @Description Remove a worker from the system
// @Tags worker
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} string
// @Router /worker/{NAME} [delete]
// @Param NAME path string false "Worker NAME"
func HandleWorkerDeleteName(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken, verbose) {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	name := vars["NAME"]

	err := database.RmWorkerName(db, name, verbose)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "")
}

// HandleWorkerStatus returns the status of a worker
// @Summary Get status of worker
// @Description Get status of worker
// @Tags worker
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} globalstructs.Worker
// @Router /worker/{NAME} [get]
// @Param NAME path string false "Worker NAME"
func HandleWorkerStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken, verbose) {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	name := vars["NAME"]

	worker, err := database.GetWorker(db, name, verbose)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(worker)
	if err != nil {
		http.Error(w, "Invalid callback body: "+err.Error()+err.Error(), http.StatusBadRequest)
		return
	}

	if verbose {
		// Print the JSON data
		log.Println("HandleWorkerStatus", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// Other functions

// ReadUserIP reads the user's IP address from the request
func ReadUserIP(r *http.Request, verbose bool) string {
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
