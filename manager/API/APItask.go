package API

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

// TASK

// @Summary Get status of tasks
// @Description Get status of tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} utils.Task
// @Router /task [get]
func HandleTaskGet(w http.ResponseWriter, r *http.Request, config utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	//get tasks
	tasks, err := database.GetTasks(db)
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
	//fmt.Println(string(jsonData))

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}

// @Summary Add a tasks
// @Description Add a tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} utils.Task
// @Router /task [post]
// @Param task body utils.Task true "Task object to create"
func HandleTaskPost(w http.ResponseWriter, r *http.Request, config utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) && incorrectOauthWorker(oauthKey, config.OauthTokenWorkers) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request utils.Task
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}

	err = database.AddTask(db, request)
	if err != nil {
		message := "Invalid worker info: " + err.Error()
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	// Handle the result as needed
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task with ID %s added", request.ID)
}

// @Summary Delete a tasks
// @Description Delete a tasks
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} string
// @Router /task/{ID} [delete]
// @Param ID path string false "task ID"
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, config utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	err := database.RmTask(db, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "")
}

// @Summary Get status of a task
// @Description Get status of a task
// @Tags task
// @Accept json
// @Produce json
// @Param Authorization header string true "OAuth Key" default(WLJ2xVQZ5TXVw4qEznZDnmEEV)
// @Success 200 {array} utils.Task
// @Router /task/{ID} [get]
// @Param ID path string false "task ID"
func HandleTaskStatus(w http.ResponseWriter, r *http.Request, config utils.ManagerConfig, db *sql.DB) {
	oauthKey := r.Header.Get("Authorization")
	if incorrectOauth(oauthKey, config.OAuthToken) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	fmt.Println("ID " + id)

	worker, err := database.GetTask(db, id)
	if err != nil {
		http.Error(w, "Invalid callback body"+err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(worker)
	if err != nil {
		http.Error(w, "Invalid callback body"+err.Error(), http.StatusBadRequest)
		return
	}

	// Print the JSON data
	//fmt.Println(string(jsonData))

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonData))
}
