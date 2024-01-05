package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/r4ulcl/nTask/manager/utils"
)

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
		log.Println("API status:", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}
