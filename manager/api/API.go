package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

// @description Handle callback from worker
// @summary Handle callback from worker
// @Tags worker
// @accept application/json
// @produce application/json
// @success 200 "OK"
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /callback [post]
func HandleCallback(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	_, okWorker := r.Context().Value("worker").(string)
	if !okWorker {
		if verbose {
			log.Println("HandleCallback: { \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	var result globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	if debug {
		log.Println(result)
		log.Println("Received result (ID: ", result.ID, " from : ", result.WorkerName, " with command: ", result.Commands)
	}

	// Update task with the worker one
	err = database.UpdateTask(db, result, verbose, debug, wg)
	if err != nil {
		if verbose {
			log.Println("HandleCallback { \"error\" : \"Error UpdateTask: " + err.Error() + "\"}")
		}
		http.Error(w, "{ \"error\" : \"Error UpdateTask: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	// force set task done
	// Set the task as running if its pending
	err = database.SetTaskStatus(db, result.ID, "done", verbose, debug, wg)
	if err != nil {
		if verbose {
			log.Println("HandleCallback { \"error\" : \"Error SetTaskStatus: " + err.Error() + "\"}")
		}
		http.Error(w, "{ \"error\" : \"Error SetTaskStatus: "+err.Error()+"\"}", http.StatusBadRequest)
		return
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
			log.Println("Error: ", err)
		}
		err = utils.SaveTaskToDisk(task, config.DiskPath, verbose, debug)
		if err != nil {
			log.Println("Error: ", err)
		}
	}

	// Handle the result as needed

	//Add 1 to Iddle thread in worker
	// add 1 when finish
	database.AddWorkerIddleThreads1(db, result.WorkerName, verbose, debug, wg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// @description Handle status from user
// @summary Handle status from user
// @Tags status
// @accept application/json
// @produce application/json
// @success 200 "OK" {object} utils.Status
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /status [get]
func HandleStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		// if not username is a worker
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	// get all data
	task, err1 := utils.GetStatusTask(db, verbose, debug)
	worker, err2 := utils.GetStatusWorker(db, verbose, debug)
	if err1 != nil || err2 != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body Marshal:"+err1.Error()+err2.Error()+"\"}", http.StatusBadRequest)
		return
	}
	status := utils.Status{
		Task:   task,
		Worker: worker,
	}

	var jsonData []byte
	jsonData, err := json.Marshal(status)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body Marshal:"+err.Error()+"\"}", http.StatusBadRequest)
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
