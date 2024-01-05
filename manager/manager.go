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
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/r4ulcl/nTask/manager/api"
	"github.com/r4ulcl/nTask/manager/database"
	"github.com/r4ulcl/nTask/manager/sshTunnel"
	"github.com/r4ulcl/nTask/manager/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

func loadManagerConfig(filename string, verbose, debug bool) (*utils.ManagerConfig, error) {
	var config utils.ManagerConfig
	if debug {
		log.Println("Loading manager config from file", filename)
	}

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

	// init WebSockets map
	config.WebSockets = make(map[string]*websocket.Conn)

	// Return nil instead of &config when error occurs
	return &config, nil
}

func loadManagerSSHConfig(filename string, verbose, debug bool) (*utils.ManagerSSHConfig, error) {
	var configSSH utils.ManagerSSHConfig
	if debug {
		log.Println("Loading manager config from file", filename)
	}

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
	err = json.Unmarshal(content, &configSSH)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return &configSSH, nil
}

func addHandleWorker(workers *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	// worker
	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerGet(w, r, config, db, verbose, debug)
	}).Methods("GET") // get workers

	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPost(w, r, config, db, verbose, debug, wg)
	}).Methods("POST") // add worker

	workers.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPostWebsocket(w, r, config, db, verbose, debug, wg, writeLock)
	})

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerDeleteName(w, r, config, db, verbose, debug, wg)
	}).Methods("DELETE") // delete worker

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerStatus(w, r, config, db, verbose, debug)
	}).Methods("GET") // check status 1 worker

}

func addHandleTask(task *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	// task
	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskGet(w, r, config, db, verbose, debug)
	}).Methods("GET") // check tasks

	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, config, db, verbose, debug, wg)
	}).Methods("POST") // Add task

	task.HandleFunc("/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, config, db, verbose, debug, wg, writeLock)
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

func StartManager(swagger bool, configFile, configSSHFile string, verifyAltName, verbose, debug bool) {
	log.Println("Running as manager...")

	// if config file empty set default
	if configFile == "" {
		configFile = "manager.conf"
	}

	config, err := loadManagerConfig(configFile, verbose, debug)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	var configSSH *utils.ManagerSSHConfig
	if configSSHFile != "" {
		configSSH, err = loadManagerSSHConfig(configSSHFile, verbose, debug)
		if err != nil {
			log.Fatal("Error loading config SSH file: ", err)
		}
	}

	// create waitGroups for DB
	var wg sync.WaitGroup
	var writeLock sync.Mutex

	// Start DB
	var db *sql.DB
	for {
		if debug {
			log.Println("Trying to connect to DB")
		}
		db, err = database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase, verbose, debug)
		if err != nil {
			log.Fatal("Error manager ConnectDB: ", err)
			if db != nil {
				db.Close()
			}
			time.Sleep(time.Second * 5)
		} else {
			break
		}
	}

	// if running set to failed
	if debug {
		log.Println("Set task running to failed")
	}
	err = database.SetTasksStatusIfRunning(db, "failed", verbose, debug, &wg)
	if err != nil {
		fmt.Println("Error SetTasksStatusIfRunning:", err)
		return
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
	go utils.VerifyWorkersLoop(db, config, verbose, debug, &wg, &writeLock)

	// manage task, routine to send task to iddle workers
	go utils.ManageTasks(config, db, verbose, debug, &wg, &writeLock)

	if configSSHFile != "" {
		go sshTunnel.StartSSH(configSSH, config.Port, verbose, debug)
	}

	router := mux.NewRouter()

	amw := authenticationMiddleware{tokenUsers: make(map[string]string), tokenWorkers: make(map[string]string)}
	amw.Populate(config)

	if swagger {
		// Start swagger endpoint
		startSwaggerWeb(router, verbose, debug)
	}

	// r.HandleFunc("/send/{recipient}", handleSendMessage).Methods("POST")

	// Status
	status := router.PathPrefix("/status").Subrouter()
	status.Use(amw.Middleware)
	status.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleStatus(w, r, config, db, verbose, debug)
	}).Methods("GET") // get callback info from task

	// Worker
	workers := router.PathPrefix("/worker").Subrouter()
	workers.Use(amw.Middleware)
	addHandleWorker(workers, config, db, verbose, debug, &wg, &writeLock)

	// Task
	task := router.PathPrefix("/task").Subrouter()
	task.Use(amw.Middleware)
	addHandleTask(task, config, db, verbose, debug, &wg, &writeLock)

	// Middleware to modify server response headers
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Modify the server response headers here
			w.Header().Set("Server", "Apache")

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	})
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
			log.Println("Error manager CertFolder: ", err)
		}
	}

	/*
		err = http.ListenAndServe(":"+config.Port, allowCORS(http.DefaultServeMux))
		if err != nil {
			log.Println("Error manager: ",err)
		}
	*/

}

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
