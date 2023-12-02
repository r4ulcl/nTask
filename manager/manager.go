// manager.go
package manager

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/NetTask/manager/API"
	"github.com/r4ulcl/NetTask/manager/database"
	"github.com/r4ulcl/NetTask/manager/utils"
)

func loadManagerConfig(filename string) (*utils.ManagerConfig, error) {
	var config utils.ManagerConfig

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading manager config file: %s\n", err)
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling manager config: %s\n", err)
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
			fmt.Println(err.Error())
		}

		// Get iddle workers
		workers, err := database.GetWorkerIddle(db)
		if err != nil {
			fmt.Println(err.Error())
		}

		// fmt.Println(len(tasks))
		// fmt.Println(len(workers))

		// if there are tasks
		if len(tasks) > 0 && len(workers) > 0 {
			// Send first to worker idle worker
			worker := workers[0]
			task := tasks[0]
			err = utils.SendAddTask(db, &worker, &task)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
		time.Sleep(time.Second * 5)
	}
}

func StartManager() {
	fmt.Println("Running as manager...")

	config, err := loadManagerConfig("manager.conf")
	if err != nil {
		fmt.Println(err)
	}

	// Start DB
	db, err := database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase)
	if err != nil {
		fmt.Println(err)
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
		API.HandleCallback(w, r, config, db)
	}).Methods("POST") // get callback info from task

	// worker
	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		API.HandleWorkerGet(w, r, config, db)
	}).Methods("GET") // get workers

	r.HandleFunc("/worker", func(w http.ResponseWriter, r *http.Request) {
		API.HandleWorkerPost(w, r, config, db)
	}).Methods("POST") // add worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		API.HandleWorkerDeleteName(w, r, config, db)
	}).Methods("DELETE") // delete worker

	r.HandleFunc("/worker/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		API.HandleWorkerStatus(w, r, config, db)
	}).Methods("GET") // check status 1 worker

	// -------------------------------------------------------------------

	// task
	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskGet(w, r, config, db)
	}).Methods("GET") // check tasks

	r.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskPost(w, r, config, db)
	}).Methods("POST") // Add task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskDelete(w, r, config, db)
	}).Methods("DELETE") // Delete task

	r.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		API.HandleTaskStatus(w, r, config, db)
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
		fmt.Println(err)
	}
	/*
		err = http.ListenAndServe(":"+config.Port, allowCORS(http.DefaultServeMux))
		if err != nil {
			fmt.Println(err)
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
