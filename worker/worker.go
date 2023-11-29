// worker.go
package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/utils"
)

type WorkerConfig struct {
	Name               string `json:"name"`
	MaxConcurrentTasks int    `json:"maxConcurrentTasks"`
	ManagerIP          string `json:"managerIP"`
	ManagerPort        string `json:"managerPort"`
	OAuthToken         string `json:"oauthToken"`
	Port               string `json:"port"`
}

type Message struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Arguments   []string `json:"arguments"`
	CallbackURL string   `json:"callbackURL"`
}

type Task struct {
	ID          string
	Module      string
	Arguments   []string
	CallbackURL string
	Status      string
	Result      string
	Goroutine   *sync.WaitGroup
}

type Status struct {
	Working   bool   `json:"working"`
	MessageID string `json:"messageID"`
}

var (
	taskList   = make(map[string]*Task)
	taskListMu sync.Mutex
	workMutex  sync.Mutex
	//maxConcurrentTasks = 1
	Working   = false
	messageID = ""
)

func loadWorkerConfig(filename string) (WorkerConfig, error) {
	var config WorkerConfig
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading worker config file: %s\n", err)
		return config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling worker config: %s\n", err)
		return config, err
	}

	return config, nil
}

func addWorker(name, port, managerIP, managerPort, oauthToken string) error {
	worker := utils.Worker{
		Name:    name,
		Port:    port,
		Working: false,
		UP:      true,
	}

	payload, _ := json.Marshal(worker)

	req, err := http.NewRequest("POST", "http://"+managerIP+":"+managerPort+"/worker", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	// Add custom headers, including the OAuth header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", oauthToken)

	// Create an HTTP client and make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return err
	}
	defer resp.Body.Close()

	//IF response is not 200 error!!
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error adding the worker %s", body)
	}

	return nil
}

func StartWorker() {
	fmt.Println("Running as worker...")

	workerConfig, err := loadWorkerConfig("worker.conf")
	if err != nil {
		fmt.Println(err)
	}

	status := Status{
		Working:   false,
		MessageID: "",
	}

	err = addWorker(workerConfig.Name, workerConfig.Port, workerConfig.ManagerIP, workerConfig.ManagerPort, workerConfig.OAuthToken)
	if err != nil {
		fmt.Println(err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		handleGetStatus(w, r, status, workerConfig)
	}).Methods("GET") //check worker status

	/*
		r.HandleFunc("/task", handletaskMessage).Methods("POST")
		r.HandleFunc("/status", handleGetStatus).Methods("GET")
		r.HandleFunc("/tasks", handleGetTasks).Methods("GET")
		r.HandleFunc("/task/{id}", handleGetTask).Methods("GET")
	*/
	http.Handle("/", r)
	err = http.ListenAndServe(":"+workerConfig.Port, nil)
	if err != nil {
		fmt.Println(err)
	}
}
