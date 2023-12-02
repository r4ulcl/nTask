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

func HandleTaskGet(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Println("TODO HandleTaskGet")
	http.Error(w, "Invalid callback body", http.StatusBadRequest)
}

// /

// /

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
