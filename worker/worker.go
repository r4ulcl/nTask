// worker.go
package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
	"github.com/r4ulcl/NetTask/worker/API"
	"github.com/r4ulcl/NetTask/worker/utils"
)

func loadWorkerConfig(filename string) (*utils.WorkerConfig, error) {
	var config utils.WorkerConfig
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading worker config file: %s\n", err)
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling worker config: %s\n", err)
		return &config, err
	}

	// if Name is empty use hostname
	if config.Name == "" {
		hostname := ""
		hostname, err = os.Hostname()
		if err != nil {
			fmt.Println("Error getting hostname:", err)
		}
		config.Name = hostname
	}

	return &config, nil
}

func StartWorker() {
	fmt.Println("Running as worker...")

	workerConfig, err := loadWorkerConfig("worker.conf")
	if err != nil {
		fmt.Println(err)
	}

	status := &globalStructs.WorkerStatus{
		Working: false,
	}

	// Loop until connects
	for {
		err = utils.AddWorker(workerConfig)
		if err != nil {
			fmt.Println(err)
		} else {
			break
		}
		time.Sleep(time.Second * 5)
	}

	r := mux.NewRouter()

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		API.HandleGetStatus(w, r, status, workerConfig)
	}).Methods("GET") // check worker status

	// Task
	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskPost(w, r, status, workerConfig)
	}).Methods("POST") // Add task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskDelete(w, r, status, workerConfig)
	}).Methods("DELETE") // delete task

	/*
		r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
			API.HandleTaskGet(w, r, status, workerConfig)
		}).Methods("GET") // check task status

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
