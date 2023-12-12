package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/modules"
	"github.com/r4ulcl/nTask/worker/utils"
)

// ------------------------------------------------------------------------------------
// -------------------------------------- Status --------------------------------------
// ------------------------------------------------------------------------------------

// HandleGetStatus handles the GET request to /status endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it returns the worker status as a JSON object.
func HandleGetStatus(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig, verbose bool) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		if verbose {
			log.Println("{ \"error\" : \"Unauthorized\" }")
		}
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		http.Error(w, "Invalid callback body"+err.Error(), http.StatusBadRequest)
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

// -------------------------------------------------------------------------------------
// ---------------------------------------- Task ---------------------------------------
// -------------------------------------------------------------------------------------

// HandleTaskPost handles the POST request to /task endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it processes the task in the background by calling the processTask function.
// It immediately responds with the ID of the task.
func HandleTaskPost(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig, verbose bool) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	var requestTask globalstructs.Task
	err := json.NewDecoder(r.Body).Decode(&requestTask)
	if err != nil {
		http.Error(w, "{ \"error\" : \"Invalid callback body: "+err.Error()+"\"}", http.StatusBadRequest)
		return
	}
	// Process TASK
	// if executing task skip and return error
	if status.IddleThreads <= 0 {
		http.Error(w, "{ \"error\" : \"The worker is working\" }", http.StatusServiceUnavailable)
		return
	}

	// Process task in background
	go processTask(status, config, &requestTask, verbose)

	// Respond immediately without waiting for the task to complete
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, requestTask.ID)
}

// HandleTaskDelete handles the DELETE request to /task/{ID} endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it stops/deletes the task with the given ID.
func HandleTaskDelete(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig, verbose bool) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	cmdID := status.WorkingIDs[id]

	// Kill the process using cmdID
	cmd := exec.Command("kill", "-9", fmt.Sprint(cmdID))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		if verbose {
			fmt.Println("Error killing process:", err)
			fmt.Println("Error details:", stderr.String())
		}
		http.Error(w, "{\"error\": \"Error killing process: "+id+"\"}", http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{\"id\": "+id+", \"status\": \"deleted\"}")
	}

}

// HandleTaskGet handles the GET request to /task/{ID} endpoint.
// It checks if the OAuth token provided by the client matches the configured token.
// If the token is valid, it returns the details of the task with the given ID.
func HandleTaskGet(w http.ResponseWriter, r *http.Request, status *globalstructs.WorkerStatus, config *utils.WorkerConfig, verbose bool) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "{ \"error\" : \"Unauthorized\" }", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	if _, exists := status.WorkingIDs[id]; exists {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{ \"id\": \""+id+"\" \n \"status\": \"running\"}")
	} else {
		http.Error(w, "{\"error\" : \"ID not found\"}", http.StatusBadRequest)

	}
}

// processTask is a helper function that processes the given task in the background.
// It sets the worker status to indicate that it is currently working on the task.
// It calls the ProcessModule function to execute the task's module.
// If an error occurs, it sets the task status to "failed".
// Otherwise, it sets the task status to "done" and assigns the output of the module to the task.
// Finally, it calls the CallbackTaskMessage function to send the task result to the configured callback endpoint.
// After completing the task, it resets the worker status to indicate that it is no longer working.
func processTask(status *globalstructs.WorkerStatus, config *utils.WorkerConfig, task *globalstructs.Task, verbose bool) {
	//Remove one from working threads
	status.IddleThreads -= 1

	if verbose {
		log.Println("Start processing task", task.ID, " workCount: ", status.IddleThreads)
	}

	err := modules.ProcessModule(task, config, status, task.ID, verbose)
	if err != nil {
		log.Println("Error ProcessModule:", err)
		task.Status = "failed"
	} else {
		task.Status = "done"
	}

	// While manager doesnt responds loop
	for {
		err = utils.CallbackTaskMessage(config, task, verbose)
		if err == nil {
			break
		} else {
			time.Sleep(time.Second * 10)
		}
	}

	//Add one from working threads
	status.IddleThreads += 1
}
