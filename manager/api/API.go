// Package api to all the nTask manager
package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/nTask/manager/utils"
)

// HandleStatus Get status summary from Manager
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
func HandleStatus(w http.ResponseWriter, r *http.Request, db *sql.DB, verbose, debug bool) {
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

// Generic handler function for fetching and encoding data
func handleEntityStatus[T any](w http.ResponseWriter, r *http.Request, db *sql.DB, verbose, debug bool, fetchDataFunc func(*sql.DB, string, bool, bool) (T, error), entityName string) {
	_, ok := r.Context().Value(utils.UsernameKey).(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	idOrName := vars[entityName]

	// Fetch the entity (task or worker)
	entity, err := fetchDataFunc(db, idOrName, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid "+entityName+" body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	// Marshal the entity data into JSON
	jsonData, err := json.Marshal(entity)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Marshal body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	if debug {
		// Print the JSON data
		log.Printf("API %s: %s", entityName, string(jsonData))
	}

	// Set the content type and write the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Use json.NewEncoder for safe encoding
	err = json.NewEncoder(w).Encode(entity)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid "+entityName+" encode body: "+err.Error()+"\"}", http.StatusBadRequest)
	}
}

func getUsername(r *http.Request, verbose, debug bool) (bool, string) {
	username, ok := r.Context().Value(utils.UsernameKey).(string)
	if debug {
		log.Println("getUsername", username)
	}
	if !ok && (debug || verbose) {
		log.Println("API { \"error\" : \"Unauthorized\" }")
	}

	return ok, username
}
