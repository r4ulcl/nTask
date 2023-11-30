package API

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os/exec"
	"time"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
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
	json.NewEncoder(w).Encode(status)
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

///

///

func processTask(status *globalStructs.WorkerStatus, config *utils.WorkerConfig, task *globalStructs.Task) {
	status.Working = true
	status.WorkingID = task.ID

	fmt.Println("Start processing task", task.ID)

	output := workAndNotify(task.ID)
	task.Output = output
	fmt.Println(task)

	utils.CallbackTaskMessage(config, task)

	status.Working = false
	status.WorkingID = ""
}

func processModule(messageID, module string, arguments []string) (string, int) {
	switch module {
	case "work1":
		workAndNotify(messageID)
		return "Task scheduled for work with an unknown duration", 0
	case "module1":
		return module1(arguments)
	case "module2":
		return module2(arguments)
	case "workList":
		if len(arguments) > 0 {
			// Simulate work with an unknown duration
			workDuration := getRandomDuration()
			time.Sleep(workDuration)
			return stringList(arguments), 0
		}
		return "", 1
	default:
		return "Unknown task", 0
	}
}

///////////////////////////////

func workAndNotify(id string) string {
	//workMutex.Lock()
	//isWorking = true
	//messageID = id
	//workMutex.Unlock()

	// Simulate work with an unknown duration
	workDuration := getRandomDuration()
	fmt.Printf("Working for %s (ID: %s)\n", workDuration.String(), id)
	time.Sleep(workDuration)

	//workMutex.Lock()
	//isWorking = false
	//messageID = ""
	//workMutex.Unlock()
	str := "Working for " + workDuration.String() + " (ID: " + id + ")"
	return str
}

func getRandomDuration() time.Duration {
	return time.Duration(rand.Intn(10)+1) * time.Second
}

func stringList(list []string) string {
	stringList := ""
	for _, item := range list {
		stringList += item + "\n"
	}
	return stringList
}

func module1(arguments []string) (string, int) {
	// Command to run the Python script
	scriptPath := "./worker/modules/module1.py"
	cmd := exec.Command("python3", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, 0
}

func module2(arguments []string) (string, int) {
	// Command to run the Bash script
	scriptPath := "./worker/modules/module2.sh"
	cmd := exec.Command("bash", append([]string{scriptPath}, arguments...)...)

	// Capture the output of the script
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Convert the output byte slice to a string
	outputString := string(output)

	return outputString, 0
}
