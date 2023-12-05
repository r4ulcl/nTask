// workerouter.go
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
	log.Println("Running as workerouter...")

	workerConfig, err := loadWorkerConfig("workerouter.conf")
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

	router := mux.NewRouter()

	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetStatus(w, r, status, workerConfig)
	}).Methods("GET") // check worker status

	// Task
	router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, status, workerConfig)
	}).Methods("POST") // Add task

	router.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, status, workerConfig)
	}).Methods("DELETE") // delete task

	/*
		router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
			api.HandleTaskGet(w, r, status, workerConfig)
		}).Methods("GET") // check task status

		router.HandleFunc("/task", handletaskMessage).Methods("POST")
		router.HandleFunc("/status", handleGetStatus).Methods("GET")
		router.HandleFunc("/tasks", handleGetTasks).Methods("GET")
		router.HandleFunc("/task/{id}", handleGetTask).Methods("GET")
	*/
	http.Handle("/", router)
	err = http.ListenAndServe(":"+workerConfig.Port, nil)
	if err != nil {
		log.Println(err)
	}
}
