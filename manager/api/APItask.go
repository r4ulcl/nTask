package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
)

// @description Get status of tasks
// @summary Get all tasks
// @Tags task
// @accept application/json
// @produce application/json
// @param ID query string false "Task ID"
// @param command query string false "Task command"
// @param name query string false "Task name"
// @param createdAt query string false "Task createdAt"
// @param updatedAt query string false "Task updatedAt"
// @param executedAt query string false "Task executedAt"
// @param status query string false "Task status" Enums(pending, running, done, failed, deleted)
// @param workerName query string false "Task workerName"
// @param username query string false "Task username"
// @param priority query string false "Task priority"
// @param callbackURL query string false "Task callbackURL"
// @param callbackToken query string false "Task callbackToken"
// @param limit query int false "limit output DB"
// @param page query int false "page output DB"
// @success 200 {array} globalstructs.Task
// @Failure 400 {object} globalstructs.Error
// @Failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /task [get]
func HandleTaskGet(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		// if not username is a worker
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	// get tasks
	tasks, err := database.GetTasks(r, db, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body GetTasks: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	var jsonData []byte
	if len(tasks) != 0 {
		jsonData, err = json.Marshal(tasks)
		if err != nil {
			http.Error(w, "{ \"error\" : \"Invalid callback body Marshal:"+err.Error()+"\"}", http.StatusBadRequest)
			return
		}
	} else {
		jsonData = []byte("[]")

	}

	if debug {
		// Print the JSON data
		log.Println(string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// @description Add a new tasks
// @summary Add a new tasks
// @Tags task
// @accept application/json
// @produce application/json
// @param task body globalstructs.TaskSwagger true "Task object to create"
// @success 200 {object} globalstructs.Task
// @Failure 400 {object} globalstructs.Error
// @Failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /task [post]
func HandleTaskPost(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	username, okUser := r.Context().Value("username").(string)
	if !okUser {
		if debug {
			log.Println("API { \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}
	if debug {
		log.Println("API HandleTaskPost", username)
	}

	var request globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		if debug {
			log.Println("API { \"error\" : \"Invalid callback body: " + err.Error() + "\"}")
		}
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	// Set Random ID
	request.ID, err = generateRandomID(30, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid ID generated: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	// set status
	request.Status = "pending"
	request.Username = username

	if request.WorkerName != "" {
		// Check if worker from user exists
		_, err := database.GetWorker(db, request.WorkerName, verbose, debug)
		if err != nil {
			http.Error(w, "{ \"error\" : \"Invalid WorkerName (not found): "+err.Error()+"\"}", http.StatusBadRequest)
			return
		}
	}

	err = database.AddTask(db, request, verbose, debug, wg)
	if err != nil {
		message := "{ \"error\" : \"Invalid task info: " + err.Error() + "\" }"
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	if verbose {
		log.Println("API Add Task to DB", request.ID)
	}

	task, err := database.GetTask(db, request.ID, verbose, debug)
	if err != nil {
		message := "{ \"error\" : \"Invalid task info: " + err.Error() + "\" }"
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	// Handle the result as needed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// @description Delete a tasks
// @summary Delete a tasks
// @Tags task
// @accept application/json
// @produce application/json
// @param ID path string true "task ID"
// @success 200 {object} globalstructs.Task
// @Failure 400 {object} globalstructs.Error
// @Failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /task/{ID} [delete]
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Username not found\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	task, err := database.GetTask(db, id, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}

	worker, err := database.GetWorker(db, task.WorkerName, verbose, debug)
	if err == nil {
		// Has a worker set, check if its running
		if task.Status == "running" {
			// If its runing send stop signal to worker
			err = utils.SendDeleteTask(db, config, &worker, &task, verbose, debug, wg, writeLock)
			if err != nil {
				http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
				return
			}
		}
	}

	// Delete task from DB
	/*err = database.RmTask(db, id, verbose, debug, wg)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}*/
	// Set task as running
	err = database.SetTaskStatus(db, id, "deleted", verbose, debug, wg)
	if err != nil {
		log.Println("Utils Error SetTaskStatus in request:", err)
	}

	// Return task with deleted status
	task.Status = "deleted"

	jsonData, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// @description Get status of a task
// @summary Get status of a task
// @Tags task
// @accept application/json
// @produce application/json
// @param ID path string true "task ID"
// @success 200 {array} globalstructs.Task
// @Failure 400 {object} globalstructs.Error
// @Failure 403 {object} globalstructs.Error
// @security ApiKeyAuth
// @router /task/{ID} [get]
func HandleTaskStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	// Access worker to update info if status running
	// get task from ID
	task, err := database.GetTask(db, id, verbose, debug)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid GetTask body: "+err.Error()+"\"}", http.StatusBadRequest)

		return
	}

	jsonData, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid Marshal body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	if debug {
		// Print the JSON data
		log.Println("API get task: ", string(jsonData))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// generateRandomID generates a random ID of the specified length
func generateRandomID(length int, verbose, debug bool) (string, error) {
	// Calculate the number of bytes needed to achieve the desired length
	numBytes := length / 2 // Since 1 byte = 2 hex characters

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Convert random bytes to hex string
	randomID := hex.EncodeToString(randomBytes)

	return randomID, nil
}
