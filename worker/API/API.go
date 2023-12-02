package API

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
	"github.com/r4ulcl/NetTask/worker/modules"
	"github.com/r4ulcl/NetTask/worker/utils"
)

// ------------------------------------------------------------------------------------
// -------------------------------------- Status --------------------------------------
// ------------------------------------------------------------------------------------

func HandleGetStatus(w http.ResponseWriter, r *http.Request, status *globalStructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

}

// -------------------------------------------------------------------------------------
// ---------------------------------------- Task ---------------------------------------
// -------------------------------------------------------------------------------------

func HandleTaskPost(w http.ResponseWriter, r *http.Request, status *globalStructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var requestTask globalStructs.Task
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

func HandleTaskDelete(w http.ResponseWriter, r *http.Request, status *globalStructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["ID"]

	fmt.Println("TODO Stop/delete", id)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, id+" deleted")
}

func HandleTaskGet(w http.ResponseWriter, r *http.Request, status *globalStructs.WorkerStatus, config *utils.WorkerConfig) {
	oauthKeyClient := r.Header.Get("Authorization")
	if oauthKeyClient != config.OAuthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	fmt.Println("TODO HandleTaskGet")
	http.Error(w, "Invalid callback body", http.StatusBadRequest)
}

// /

// /

func processTask(status *globalStructs.WorkerStatus, config *utils.WorkerConfig, task *globalStructs.Task) {
	status.Working = true
	status.WorkingID = task.ID

	fmt.Println("Start processing task", task.ID)

	output, err := processModule(task)
	if err != nil {
		fmt.Println("Error:", err)
		task.Status = "failed"
	} else {
		task.Status = "done"
	}
	task.Output = output

	utils.CallbackTaskMessage(config, task)

	status.Working = false
	status.WorkingID = ""
}

func processModule(task *globalStructs.Task) (string, error) {
	messageID := task.ID
	module := task.Module
	arguments := task.Args
	switch module {
	case "work1":
		return modules.WorkAndNotify(messageID)
	case "module1":
		return modules.Module1(arguments)
	case "module2":
		return modules.Module2(arguments)
	case "workList":
		if len(arguments) > 0 {
			// Simulate work with an unknown duration
			workDuration := modules.GetRandomDuration()
			time.Sleep(workDuration)
			return modules.StringList(arguments), nil
		}
		return "", nil
	default:
		return "Unknown task", fmt.Errorf("unknown task")
	}
}
