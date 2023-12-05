package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
// @Produce []json
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
func HandleTaskGet(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// get tasks
	tasks, err := database.GetTasks(w, r, db)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(tasks)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}

	// Print the JSON data
	// log.Println(string(jsonData))

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
// @Param task body globalstructs.Task true "Task object to create"
func HandleTaskPost(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) && incorrectOauthWorker(oauthKey, config.OauthTokenWorkers) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}

	// Set Random ID
	request.ID, err = generateRandomID(30)
	if err != nil {
		http.Error(w, "Invalid ID generated", http.StatusBadRequest)
		return
	}

	// set status
	request.Status = "pending"

	err = database.AddTask(db, request)
	if err != nil {
		message := "Invalid task info: " + err.Error()
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	// Handle the result as needed
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task with ID %s added", request.ID)
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
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	task, err := database.GetTask(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	worker, err := database.GetWorker(db, task.WorkerName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = utils.SendDeleteTask(db, &worker, &task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = database.RmTask(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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
func HandleTaskStatus(w http.ResponseWriter, r *http.Request, config *utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	// Access worker to update info if status running
	// get task from ID
	task, err := database.GetTask(db, id)
	if err != nil {
		http.Error(w, "Invalid GetTask body"+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(task)
	if err != nil {
		http.Error(w, "Invalid Marshal body"+err.Error(), http.StatusBadRequest)
		return
	}

	// Print the JSON data
	// log.Println(string(jsonData))

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// generateRandomID generates a random ID of the specified length
func generateRandomID(length int) (string, error) {
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
