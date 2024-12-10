package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/r4ulcl/nTask/manager/utils"
)

// @description Get status summary from Manager
// @summary Get status summary from Manager
// @Tags status
// @accept application/json
// @produce application/json
// @success 200 "OK" {object} utils.Status
// @failure 400 {object} globalstructs.Error
// @failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /status [get]
func HandleStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	_, ok := r.Context().Value(utils.UsernameKey).(string)
	if !ok {
		// if not username is a worker
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	// get all data
	tasks, err1 := utils.GetStatusTask(db, verbose, debug)
	workers, err2 := utils.GetStatusWorker(db, verbose, debug)
	if err1 != nil || err2 != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body Marshal:"+err1.Error()+err2.Error()+"\"}", http.StatusBadRequest)
		return
	}
	status := utils.Status{
		Task:   tasks,
		Worker: workers,
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
	// Use json.NewEncoder for safe encoding
	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid status encode body:"+err.Error()+"\"}", http.StatusBadRequest)
	}
}
