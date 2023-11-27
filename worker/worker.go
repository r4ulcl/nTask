// slave.go
package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
	"math/rand"
	"os/exec"

	"github.com/gorilla/mux"
)

type WorkerConfig struct {
	MaxConcurrentTasks int    `json:"maxConcurrentTasks"`
	OAuthToken         string `json:"oauthToken"`
	Port			   string `json:"port"`
}

type Message struct {
	ID           string   `json:"id"`
	Module       string   `json:"module"`
	Arguments    []string `json:"arguments"`
	CallbackURL  string   `json:"callbackURL"`
}

type Task struct {
	ID           string
	Module       string
	Arguments    []string 
	CallbackURL  string
	Status       string
	Goroutine    *sync.WaitGroup
}

type Status struct {
	IsWorking    bool   `json:"isWorking"`
	RemainingSec int    `json:"remainingSec"`
	MessageID    string `json:"messageID"`
}


var (
	taskList   = make(map[string]*Task)
	taskListMu sync.Mutex
	workMutex  sync.Mutex
	//maxConcurrentTasks = 1
	semaphoreCh  = make(chan struct{}, 1)
	isWorking  = false
	messageID    = ""
	oauthToken   = "your_oauth_token" // Replace with your actual OAuth token
	port 		= "8081"
)

func loadWorkerConfig(filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading worker config file: %s\n", err)
		return 
	}

	var config WorkerConfig
	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling worker config: %s\n", err)
		return 
	}

	semaphoreCh  = make(chan struct{}, config.MaxConcurrentTasks)
	oauthToken = config.OAuthToken
	port = config.Port


	return
}


func handleReceiveMessage(w http.ResponseWriter, r *http.Request) {
	oauthKey := r.Header.Get("Authorization")
	if oauthKey != oauthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var message Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received message (ID: %s): %s\n", message.ID, message.Module)

	// Create a new Task
	task := &Task{
		ID:          message.ID,
		Module:	 	 message.Module,
		Arguments:	 message.Arguments,
		CallbackURL: message.CallbackURL,
		Status: 	 "pending",
		Goroutine:   &sync.WaitGroup{},
	}

	// Add the task to the list
	taskListMu.Lock()
	taskList[message.ID] = task
	taskListMu.Unlock()

	// Start a new goroutine for the task
	task.Goroutine.Add(1)
	go processTask(message, task)

	// Respond immediately without waiting for the task to complete
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Message received")
}

func processTask(message Message, task *Task) {
	defer func() {
		task.Goroutine.Done()
		// Release a slot in the semaphore when the task is done
		<-semaphoreCh
	}()

	// Acquire a slot from the semaphore
	semaphoreCh <- struct{}{}

	//Set task status
	task.Status = "Working"

	workMutex.Lock()
	isWorking = true
	workMutex.Unlock()

	// Process the module in the task
	m, err := processModule(message.Module, message.Arguments)
	if err != 0 {
		fmt.Printf("Failed to run module")
	}

	workMutex.Lock()
	isWorking = false
	workMutex.Unlock()

	//Set task status
	task.Status = "Done"

	// Remove the task from the list
	taskListMu.Lock()
	delete(taskList, task.ID)
	taskListMu.Unlock()

	// Notify the master about the result with the unique ID
	result := Message{
		ID:          task.ID,
		Module:      m,
		CallbackURL: task.CallbackURL,
	}
	payload, _ := json.Marshal(result)
	http.Post(task.CallbackURL, "application/json", bytes.NewBuffer(payload))
}

func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	oauthKey := r.Header.Get("Authorization")
	if oauthKey != oauthToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	workMutex.Lock()
	defer workMutex.Unlock()

	status := Status{
		IsWorking:    isWorking,
		MessageID:    messageID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleGetTasks(w http.ResponseWriter, r *http.Request) {
	taskListMu.Lock()
	defer taskListMu.Unlock()

	tasks := taskList

	//TODO if empty

	responseJSON, err := json.Marshal(tasks)
	if err != nil {
		http.Error(w, "Error encoding tasks to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func processModule(module string, arguments []string) (string, int) {
	switch module {
	case "work1":
		workAndNotify(1, messageID)
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

func workAndNotify(seconds int, id string) {
	//workMutex.Lock()
	isWorking = true
	messageID = id
	//workMutex.Unlock()

	// Simulate work with an unknown duration
	workDuration := getRandomDuration()
	fmt.Printf("Working for %s (ID: %s)\n", workDuration.String(), id)
	time.Sleep(workDuration)

	//workMutex.Lock()
	isWorking = false
	messageID = ""
	//workMutex.Unlock()
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


func StartWorker() {
	fmt.Println("Running as worker...")

	loadWorkerConfig("worker.conf")

	r := mux.NewRouter()
	r.HandleFunc("/receive", handleReceiveMessage).Methods("POST")
	r.HandleFunc("/status", handleGetStatus).Methods("GET")
	r.HandleFunc("/tasks", handleGetTasks).Methods("GET")

	http.Handle("/", r)
	http.ListenAndServe(":" + port, nil)
}
