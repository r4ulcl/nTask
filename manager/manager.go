// manager.go
package manager

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/api"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

func loadManagerConfig(filename string, verbose bool) (*utils.ManagerConfig, error) {
	var config utils.ManagerConfig

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}

func manageTasks(config *utils.ManagerConfig, db *sql.DB, verbose bool) error {
	// infinite loop eecuted with go routine
	for {
		// Get all tasks in order and if priority
		tasks, err := database.GetTasksPending(db, verbose)
		if err != nil {
			log.Println(err.Error())
		}

		// Get iddle workers
		workers, err := database.GetWorkerIddle(db, verbose)
		if err != nil {
			log.Println(err.Error())
		}

		// log.Println(len(tasks))
		// log.Println(len(workers))

		// if there are tasks
		if len(tasks) > 0 && len(workers) > 0 {
			if verbose {
				log.Println("len(tasks)", len(tasks))
				log.Println("len(workers)", len(workers))
			}
			for _, task := range tasks {
				for _, worker := range workers {
					// if WorkerName not send or set this worker, just sendAddTask
					if task.WorkerName == "" || task.WorkerName == worker.Name {
						err = utils.SendAddTask(db, &worker, &task, verbose)
						if err != nil {
							log.Println(err.Error())
							time.Sleep(time.Second * 1)
						}
					}
				}	
			}
		} else {
			// only wait if not tasks or no workers
			time.Sleep(time.Second * 1)
		}

	}
}

func addHandleWorker(router *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	// worker
	router.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerGet(w, r, config, db, verbose)
	}).Methods("GET") // get workers

	router.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPost(w, r, config, db, verbose)
	}).Methods("POST") // add worker

	router.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerDeleteName(w, r, config, db, verbose)
	}).Methods("DELETE") // delete worker

	router.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerStatus(w, r, config, db, verbose)
	}).Methods("GET") // check status 1 worker
}

func addHandleTask(router *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose bool) {
	// task
	router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskGet(w, r, config, db, verbose)
	}).Methods("GET") // check tasks

	router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, config, db, verbose)
	}).Methods("POST") // Add task

	router.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, config, db, verbose)
	}).Methods("DELETE") // Delete task

	router.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskStatus(w, r, config, db, verbose)
	}).Methods("GET") // get status task

}

func startSwaggerWeb(router *mux.Router, verbose bool) {
	// Serve Swagger UI at /swagger
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"), // URL to the swagger.json file
	))

	// Serve Swagger JSON at /swagger/doc.json
	router.HandleFunc("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	}).Methods("GET")
}

func StartManager(swagger bool, configFile string, verbose bool) {
	log.Println("Running as manager...")

	// if config file empty set default
	if configFile == "" {
		configFile = "manager.conf"
	}

	config, err := loadManagerConfig(configFile, verbose)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	// Start DB
	var db *sql.DB
	for {
		db, err = database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase, verbose)
		if err != nil {
			log.Println(err)
			db.Close()
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	// verify status workers infinite
	go utils.VerifyWorkersLoop(db, verbose)

	// manage task, routine to send task to iddle workers
	go manageTasks(config, db, verbose)

	router := mux.NewRouter()

	if swagger {
		// Start swagger endpoint
		startSwaggerWeb(router, verbose)
	}

	// r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")

	// CallBack
	router.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		api.HandleCallback(w, r, config, db, verbose)
	}).Methods("POST") // get callback info from task

	// Worker
	addHandleWorker(router, config, db, verbose)

	// Task
	addHandleTask(router, config, db, verbose)

	http.Handle("/", router)
	err = http.ListenAndServe(":"+config.Port, nil)
	if err != nil {
		log.Println(err)
	}

	/*
		err = http.ListenAndServe(":"+config.Port, allowCORS(http.DefaultServeMux))
		if err != nil {
			log.Println(err)
		}
	*/

}

/*
// allowCORS is a middleware function that adds CORS headers to the response.
func allowCORS(handler http.Handler, verbose bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Add Authorization header

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call the next handler in the chain
		handler.ServeHTTP(w, r)
	})
}
*/
