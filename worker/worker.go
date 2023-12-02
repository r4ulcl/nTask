// worker.go
package worker

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
	"github.com/r4ulcl/NetTask/worker/api"
	"github.com/r4ulcl/NetTask/worker/utils"
)

func loadWorkerConfig(filename string) (*utils.WorkerConfig, error) {
	var config utils.WorkerConfig
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("Error reading worker config file: ", err)
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Println("Error unmarshalling worker config: ", err)
		return &config, err
	}

	// if Name is empty use hostname
	if config.Name == "" {
		hostname := ""
		hostname, err = os.Hostname()
		if err != nil {
			log.Println("Error getting hostname:", err)
		}
		config.Name = hostname
	}

	// Print the values from the struct
	log.Println("Name:", config.Name)
	log.Println("Tasks:")
	for module, exec := range config.Modules {
		log.Printf("  Module: %s, Exec: %s\n", module, exec)
	}

	return &config, nil
}

func StartWorker() {
	log.Println("Running as worker...")

	workerConfig, err := loadWorkerConfig("worker.conf")
	if err != nil {
		log.Println(err)
	}

	status := &globalstructs.WorkerStatus{
		Working: false,
	}

	// Loop until connects
	for {
		err = utils.AddWorker(workerConfig)
		if err != nil {
			log.Println(err)
		} else {
			break
		}
		time.Sleep(time.Second * 5)
	}

	r := mux.NewRouter()

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetStatus(w, r, status, workerConfig)
	}).Methods("GET") // check worker status

	// Task
	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, status, workerConfig)
	}).Methods("POST") // Add task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, status, workerConfig)
	}).Methods("DELETE") // delete task

	/*
		r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
			api.HandleTaskGet(w, r, status, workerConfig)
		}).Methods("GET") // check task status

		r.HandleFunc("/task", handletaskMessage).Methods("POST")
		r.HandleFunc("/status", handleGetStatus).Methods("GET")
		r.HandleFunc("/tasks", handleGetTasks).Methods("GET")
		r.HandleFunc("/task/{id}", handleGetTask).Methods("GET")
	*/
	http.Handle("/", r)
	err = http.ListenAndServe(":"+workerConfig.Port, nil)
	if err != nil {
		log.Println(err)
	}
}
