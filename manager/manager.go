// manager.go
package manager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/r4ulcl/nTask/manager/api"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

func loadManagerConfig(filename string, verbose, debug bool) (*utils.ManagerConfig, error) {
	var config utils.ManagerConfig

	// Validate filename
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist")
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Use specific error message for json.Unmarshal failure
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	// Return nil instead of &config when error occurs
	return &config, nil
}

func manageTasks(config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	// infinite loop eecuted with go routine
	for {
		// Get all tasks in order and if priority
		tasks, err := database.GetTasksPending(db, verbose, debug)
		if err != nil {
			log.Println(err.Error())
		}

		// Get iddle workers
		workers, err := database.GetWorkerIddle(db, verbose, debug)
		if err != nil {
			log.Println(err.Error())
		}

		//log.Println(len(tasks))
		//log.Println(len(workers))

		// if there are tasks
		if len(tasks) > 0 && len(workers) > 0 {
			if debug {
				log.Println("len(tasks)", len(tasks))
				log.Println("len(workers)", len(workers))
			}
			for _, task := range tasks {
				for _, worker := range workers {
					// if WorkerName not send or set this worker, just sendAddTask
					if task.WorkerName == "" || task.WorkerName == worker.Name {
						err = utils.SendAddTask(db, config, &worker, &task, verbose, debug)
						if err != nil {
							log.Println("Error SendAddTask", err.Error())
							//time.Sleep(time.Second * 1)
							break
						}
					}
				}
				// Update iddle workers after loop all
				workers, err = database.GetWorkerIddle(db, verbose, debug)
				if err != nil {
					log.Println(err.Error())
				}
				// If no workers just start again
				if len(workers) == 0 {
					break
				}
			}
		}
	}
}

func addHandleWorker(workers *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	// worker
	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerGet(w, r, config, db, verbose, debug)
	}).Methods("GET") // get workers

	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPost(w, r, config, db, verbose, debug)
	}).Methods("POST") // add worker

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerDeleteName(w, r, config, db, verbose, debug)
	}).Methods("DELETE") // delete worker

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerStatus(w, r, config, db, verbose, debug)
	}).Methods("GET") // check status 1 worker
}

func addHandleTask(task *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool) {
	// task
	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskGet(w, r, config, db, verbose, debug)
	}).Methods("GET") // check tasks

	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, config, db, verbose, debug)
	}).Methods("POST") // Add task

	task.HandleFunc("/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, config, db, verbose, debug)
	}).Methods("DELETE") // Delete task

	task.HandleFunc("/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskStatus(w, r, config, db, verbose, debug)
	}).Methods("GET") // get status task

}

func startSwaggerWeb(router *mux.Router, verbose, debug bool) {
	// Serve Swagger UI at /swagger
	//swagger := router.PathPrefix("/swagger").Subrouter()
	router.PathPrefix("/swagger").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"), // URL to the swagger.json file
	))

	// Serve Swagger JSON at /swagger/doc.json
	router.HandleFunc("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	}).Methods("GET")

	if verbose {
		log.Println("Configure swagger docs in /swagger/")
	}
}

func StartManager(swagger bool, configFile string, verifyAltName, verbose, debug bool) {
	log.Println("Running as manager...")

	// if config file empty set default
	if configFile == "" {
		configFile = "manager.conf"
	}

	config, err := loadManagerConfig(configFile, verbose, debug)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	// Start DB
	var db *sql.DB
	for {
		db, err = database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase, verbose, debug)
		if err != nil {
			log.Println("Error manager: ", err)
			db.Close()
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	// Create an HTTP client with the custom TLS configuration
	if config.CertFolder != "" {
		clientHTTP, err := utils.CreateTLSClientWithCACert(config.CertFolder+"/ca-cert.pem", verifyAltName, verbose, debug)
		if err != nil {
			fmt.Println("Error creating HTTP client:", err)
			return
		}
		config.ClientHTTP = clientHTTP
	} else {
		config.ClientHTTP = &http.Client{}
	}

	// verify status workers infinite
	go utils.VerifyWorkersLoop(db, config, verbose, debug)

	// manage task, routine to send task to iddle workers
	go manageTasks(config, db, verbose, debug)

	router := mux.NewRouter()

	amw := authenticationMiddleware{tokenUsers: make(map[string]string), tokenWorkers: make(map[string]string)}
	amw.Populate(config)

	if swagger {
		// Start swagger endpoint
		startSwaggerWeb(router, verbose, debug)
	}

	// r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")

	// CallBack
	callback := router.PathPrefix("/callback").Subrouter()
	callback.Use(amw.Middleware)
	callback.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleCallback(w, r, config, db, verbose, debug)
	}).Methods("POST") // get callback info from task

	// Worker
	workers := router.PathPrefix("/worker").Subrouter()
	workers.Use(amw.Middleware)
	addHandleWorker(workers, config, db, verbose, debug)

	// Task
	task := router.PathPrefix("/task").Subrouter()
	task.Use(amw.Middleware)
	addHandleTask(task, config, db, verbose, debug)

	//router.Use(amw.Middleware)

	http.Handle("/", router)

	// Set string for the port
	addr := fmt.Sprintf(":%s", config.Port)
	if verbose {
		log.Println("Port", config.Port)
	}

	// if there is cert is HTTPS
	if config.CertFolder != "" {
		log.Fatal(http.ListenAndServeTLS(addr, config.CertFolder+"/cert.pem", config.CertFolder+"/key.pem", router))
	} else {
		err = http.ListenAndServe(addr, nil)
		if err != nil {
			log.Println("Error manager: ", err)
		}
	}

	/*
		err = http.ListenAndServe(":"+config.Port, allowCORS(http.DefaultServeMux))
		if err != nil {
			log.Println("Error manager: ",err)
		}
	*/

}

/*
// allowCORS is a middleware function that adds CORS headers to the response.
func allowCORS(handler http.Handler, verbose, debug bool) http.Handler {
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

// Define our struct
type authenticationMiddleware struct {
	tokenUsers   map[string]string
	tokenWorkers map[string]string
}

// Initialize it somewhere
func (amw *authenticationMiddleware) Populate(config *utils.ManagerConfig) {
	// the key is the token instead of user
	for k, v := range config.Users {
		amw.tokenUsers[v] = k
	}
	for k, v := range config.Workers {
		amw.tokenWorkers[v] = k
	}
}

// Middleware function, which will be called for each request
func (amw *authenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/swagger/") && r.URL.Path != "/docs/swagger.json" {
			token := r.Header.Get("Authorization")
			user, foundUser := amw.tokenUsers[token]
			worker, foundWorker := amw.tokenWorkers[token]
			if foundUser {
				// We found the token in our map
				// Add the username to the request context
				ctx := context.WithValue(r.Context(), "username", user)

				// Pass down the request with the updated context to the next middleware (or final handler)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else if foundWorker {
				// We found the token in our map
				// Add the username to the request context
				ctx := context.WithValue(r.Context(), "worker", worker)

				// Pass down the request with the updated context to the next middleware (or final handler)
				next.ServeHTTP(w, r.WithContext(ctx))

			} else {
				// Write an error and stop the handler chain
				http.Error(w, "{ \"error\" : \"Forbidden\" }", http.StatusForbidden)

			}
		}
	})
}
