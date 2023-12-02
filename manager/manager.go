// manager.go
package manager

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/api"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

func loadManagerConfig(filename string) (*utils.ManagerConfig, error) {
	var config utils.ManagerConfig

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("Error reading manager config file: ", err)
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Println("Error unmarshalling manager config: ", err)
		return &config, err
	}

	return &config, nil
}

func manageTasks(config *utils.ManagerConfig, db *sql.DB) {
	// infinite loop eecuted with go routine
	for {

		// Get all tasks in order and if priority
		tasks, err := database.GetTasksPending(db)
		if err != nil {
			log.Println(err.Error())
		}

		// Get iddle workers
		workers, err := database.GetWorkerIddle(db)
		if err != nil {
			log.Println(err.Error())
		}

		// log.Println(len(tasks))
		// log.Println(len(workers))

		// if there are tasks
		if len(tasks) > 0 && len(workers) > 0 {
			// Send first to worker idle worker
			worker := workers[0]
			task := tasks[0]
			err = utils.SendAddTask(db, &worker, &task)
			if err != nil {
				log.Println(err.Error())
			}
		} else {
			//only wait if not tasks or no workers
			time.Sleep(time.Second * 1)
		}

	}
}

func StartManager() {
	log.Println("Running as manager...")

	config, err := loadManagerConfig("manager.conf")
	if err != nil {
		log.Println(err)
	}

	// Start DB
	db, err := database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase)
	if err != nil {
		log.Println(err)
	}

	// verify status workers infinite
	go utils.VerifyWorkersLoop(db)

	// manage task, routine to send task to iddle workers
	go manageTasks(config, db)

	r := mux.NewRouter()

	// Serve Swagger UI at /swagger
	// Serve Swagger UI at /swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"), // URL to the swagger.json file
	))

	// Serve Swagger JSON at /swagger/doc.json
	r.HandleFunc("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	}).Methods("GET")

	// r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")

	// CallBack
	r.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		api.HandleCallback(w, r, config, db)
	}).Methods("POST") // get callback info from task

	// worker
	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerGet(w, r, config, db)
	}).Methods("GET") // get workers

	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPost(w, r, config, db)
	}).Methods("POST") // add worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerDeleteName(w, r, config, db)
	}).Methods("DELETE") // delete worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerStatus(w, r, config, db)
	}).Methods("GET") // check status 1 worker

	// -------------------------------------------------------------------

	// task
	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskGet(w, r, config, db)
	}).Methods("GET") // check tasks

	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, config, db)
	}).Methods("POST") // Add task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, config, db)
	}).Methods("DELETE") // Delete task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskStatus(w, r, config, db)
	}).Methods("GET") // get status task

	// r.HandleFunc("/task/{ID}", handletasktop).Methods("PATCH")

	// -------------------------------------------------------------------

	/*
		// vuln
		r.HandleFunc("/vuln/", handleDummy).Methods("GET")
		r.HandleFunc("/vuln/add", handleDummy).Methods("POST")
		r.HandleFunc("/vuln/rm", handleDummy).Methods("POST")
		r.HandleFunc("/vuln/info/{id}", handleDummy).Methods("GET")

		// -------------------------------------------------------------------

		// Scope

		// -------------------------------------------------------------------

		// Asset
	*/

	http.Handle("/", r)
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
func allowCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http:// localhost:8000")
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
