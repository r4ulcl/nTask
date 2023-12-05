package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/worker/modules"
	"github.com/r4ulcl/NetTask/worker/utils"
)

// ------------------------------------------------------------------------------------
// -------------------------------------- Status --------------------------------------
// ------------------------------------------------------------------------------------

// HandleGetStatus handles the GET request to /status endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it returns the worker status as a JSON object.
func HandleGetStatus(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		log.Fatalln("Error encoding status", err)
	}
}

// -------------------------------------------------------------------------------------
// ---------------------------------------- Task ---------------------------------------
// -------------------------------------------------------------------------------------

// HandleTaskPost handles the POST request to /task endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it processes the task in the background by calling the processTask function.
// It immediately responds with the ID of the task.
func HandleTaskPost(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var requestTask globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&requestTask)
	if err != nil {
		http.Error(w, "Invalid callback body", http.StatusBadRequest)
		return
	}
	// Process TASK
	// if executing task skip and return error
	if status.Working {
		http.Error(w, "The worker is working", http.StatusServiceUnavailable)
		return
	}

	// Process task in background
	go processTask(status, config, &requestTask)

	// Respond immediately without waiting for the task to complete
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, requestTask.ID)
}

// HandleTaskDelete handles the DELETE request to /task/{ID} endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it stops/deletes the task with the given ID.
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	log.Println("TODO Stop/delete", id)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, id+" deleted")
}

// HandleTaskGet handles the GET request to /task/{ID} endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it returns the details of the task with the given ID.
func HandleTaskGet(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Println("TODO HandleTaskGet")
	http.Error(w, "Invalid callback body", http.StatusBadRequest)
}

// processTask is a helper function that processes the given task in the background.
// It sets the worker status to indicate that it is currently working on the task.
// It calls the ProcessModule function to execute the task's module.
// If an error occurs, it sets the task status to "failed".
// Otherwise, it sets the task status to "done" and assigns the output of the module to the task.
// Finally, it calls the CallbackTaskMessage function to send the task result to the configured callback endpoint.
// After completing the task, it resets the worker status to indicate that it is no longer working.
func processTask(status *globalstructs.WorkerStatus, config *utils.WorkerConfig, task *globalstructs.Task) {
	status.Working = true
	status.WorkingID = task.ID

	log.Println("Start processing task", task.ID)

	output, err := modules.ProcessModule(task, config)
	if err != nil {
		log.Println("Error:", err)
		task.Status = "failed"
	} else {
		task.Status = "done"
	}
	task.Output = output

	utils.CallbackTaskMessage(config, task)

	status.Working = false
	status.WorkingID = ""
}
