package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

// HandleTaskGet - Get all tasks
// @Summary Get all tasks
// @Description Get status of tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} globalstructs.Task
// @Router /task [get]
// @Param ID query string false "Task ID"
// @Param module query string false "Task module"
// @Param args query string false "Task args"
// @Param createdAt query string false "Task createdAt"
// @Param updatedAt query string false "Task updatedAt"
// @Param status query string false "Task status" Enum(pending, running, done, failed) Example(pending)
// @Param workerName query string false "Task workerName"
// @Param output query string false "Task output"
// @Param priority query bool false "Task priority"
func HandleTaskGet(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Username not found\" }", http.StatusUnauthorized)
		return
	}

	// get tasks
	tasks, err := database.GetTasks(w, r, db, verbose)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body GetTasks: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(tasks)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body Marshal:"+err.Error()+"\"}", http.StatusBadRequest)
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

// HandleTaskPost - Add a new tasks
// @Summary Add a new tasks
// @Description Add a new tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} globalstructs.Task
// @Router /task [post]
// @Param task body globalstructs.TaskSwagger true "Task object to create"
func HandleTaskPost(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	username, okUser := r.Context().Value("username").(string)
	if !okUser {
		if verbose {
			log.Println("{ \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}
	log.Println("HandleTaskPost", username)

	var request globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		if verbose {
			log.Println("{ \"error\" : \"Invalid callback body: " + err.Error() + "\"}")
		}
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}

	// Set Random ID
	request.ID, err = generateRandomID(30, verbose)
	if err != nil {
		http.Error(w, "Invalid ID generated", http.StatusBadRequest)
		return
	}

	// set status
	request.Status = "pending"
	request.Username = username

	err = database.AddTask(db, request, verbose)
	if err != nil {
		message := "{ \"error\" : \"Invalid task info: " + err.Error() + "\" }"
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	task, err := database.GetTask(db, request.ID, verbose)
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

// HandleTaskDelete - Delete a tasks
// @Summary Delete a tasks
// @Description Delete a tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} string
// @Router /task/{ID} [delete]
// @Param ID path string false "task ID"
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Username not found\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	task, err := database.GetTask(db, id, verbose)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}

	worker, err := database.GetWorker(db, task.WorkerName, verbose)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}

	err = utils.SendDeleteTask(db, &worker, &task, verbose)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}

	err = database.RmTask(db, id, verbose)
	if err != nil {
		http.Error(w, "{ \"error\" : \""+err.Error()+"\" }", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "")
}

// HandleTaskStatus - Get status of a task
// @Summary Get status of a task
// @Description Get status of a task
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} globalstructs.Task
// @Router /task/{ID} [get]
// @Param ID path string false "task ID"
func HandleTaskStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	_, ok := r.Context().Value("username").(string)
	if !ok {
		http.Error(w, "{ \"error\" : \"Username not found\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	// Access worker to update info if status running
	// get task from ID
	task, err := database.GetTask(db, id, verbose)
	if err != nil {
		http.Error(w, "Invalid GetTask body"+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "Invalid Marshal body"+err.Error(), http.StatusBadRequest)
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

// generateRandomID generates a random ID of the specified length
func generateRandomID(length int, verbose bool) (string, error) {
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
