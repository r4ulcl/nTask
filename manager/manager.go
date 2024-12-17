// Package manager with manager main data
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
	"github.com/r4ulcl/nTask/manager/cloud"
	"github.com/r4ulcl/nTask/manager/database"
	sshtunnel "github.com/r4ulcl/nTask/manager/sshTunnel"
	"github.com/r4ulcl/nTask/manager/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Helper function to load and parse JSON config files
func loadConfigFile[T any](filename string, verbose, debug bool, configType string) (*T, error) {
	if debug || verbose {
		log.Printf("Manager Loading %s config from file: %s", configType, filename)
	}

	// Validate filename
	if filename == "" {
		return nil, errors.New("filename cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filename)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Unmarshal the content into the generic type T
	var config T
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON for %s: %w", configType, err)
	}

	return &config, nil
}

// Specific function to load nTask config
func loadManagerConfig(filename string, verbose, debug bool) (*utils.ManagerConfig, error) {
	configFile, err := loadConfigFile[utils.ManagerConfig](filename, verbose, debug, "nTask")
	if err != nil {
		return nil, err
	}
	// init WebSockets map
	configFile.WebSockets = make(map[string]*websocket.Conn)
	return configFile, nil
}

// Specific function to load SSH config
func loadManagerSSHConfig(filename string, verbose, debug bool) (*utils.ManagerSSHConfig, error) {
	return loadConfigFile[utils.ManagerSSHConfig](filename, verbose, debug, "SSH")
}

// Specific function to load Cloud config
func loadManagerCloudConfig(filename string, verbose, debug bool) (*utils.ManagerCloudConfig, error) {
	return loadConfigFile[utils.ManagerCloudConfig](filename, verbose, debug, "Cloud")
}

func addHandleWorker(workers *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	// worker
	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerGet(w, r, db, verbose, debug)
	}).Methods("GET") // get workers

	workers.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPost(w, r, db, verbose, debug, wg)
	}).Methods("POST") // add worker

	workers.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerPostWebsocket(w, r, config, db, verbose, debug, wg, writeLock)
	})

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerDeleteName(w, r, db, verbose, debug, wg)
	}).Methods("DELETE") // delete worker

	workers.HandleFunc("/{NAME}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleWorkerStatus(w, r, db, verbose, debug)
	}).Methods("GET") // check status 1 worker

}

func addHandleTask(task *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex) {
	// task
	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskGet(w, r, db, verbose, debug)
	}).Methods("GET") // check tasks

	task.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, db, verbose, debug, wg)
	}).Methods("POST") // Add task

	task.HandleFunc("/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, config, db, verbose, debug, wg, writeLock)
	}).Methods("DELETE") // Delete task

	task.HandleFunc("/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskStatus(w, r, db, verbose, debug)
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

	if verbose || debug {
		log.Println("Manager Configure swagger docs in /swagger/")
	}
}

// StartManager main function to start manager
// StartManager initializes and starts the manager application
func StartManager(swagger bool, configFile, configSSHFile, configCloudFile string, verifyAltName, verbose, debug bool) {
	log.Println("Manager Running as manager...")

	var wg sync.WaitGroup
	var writeLock sync.Mutex

	// Load configurations
	config, err := loadManagerConfigurations(configFile, verbose, debug)
	if err != nil {
		log.Println("Error loadManagerConfigurations")
		return
	}
	configSSH, err := loadSSHConfiguration(configSSHFile, verbose, debug)
	if err != nil {
		log.Println("Error loadSSHConfiguration")
	}
	configCloud, err := loadCloudConfiguration(configCloudFile, verbose, debug)
	if err != nil {
		log.Println("Error loadCloudConfiguration")
	}

	// Connect to database
	db := connectToDatabase(config, debug)
	defer db.Close()

	// Handle initial task status updates
	setInitialTaskStatus(db, verbose, debug)

	// Initialize HTTP client
	if config != nil {
		initializeHTTPClient(config, verifyAltName, verbose, debug)
		startBackgroundTask(db, config, &wg, &writeLock, verbose, debug)
		setupAndStartServers(swagger, config, db, &wg, &writeLock, verbose, debug)
	}

	// Start SSH background task
	if configSSH != nil {
		startSSHBackgroundTask(configSSH, config, verbose, debug)
	}

	if configCloud != nil {
		processCloudConfiguration(configCloud, configSSH, verbose, debug)
	}
}

func loadManagerConfigurations(configFile string, verbose, debug bool) (*utils.ManagerConfig, error) {
	if configFile == "" {
		configFile = "manager.conf"
	}

	config, err := loadManagerConfig(configFile, verbose, debug)
	if err != nil {
		return nil, fmt.Errorf("Error loading config file")
	}

	// Load default values
	if config.APIIdleTimeout <= 0 {
		config.APIIdleTimeout = 60
	}
	if config.APIReadTimeout <= 0 {
		config.APIIdleTimeout = 60
	}
	if config.APIWriteTimeout <= 0 {
		config.APIIdleTimeout = 60
	}
	if config.APIIdleTimeout <= 0 {
		config.APIIdleTimeout = 60
	}
	if config.HTTPPort <= 0 {
		config.APIIdleTimeout = 8080
	}
	if config.HTTPSPort <= 0 {
		config.APIIdleTimeout = 8443
	}
	if config.StatusCheckSeconds <= 0 {
		config.StatusCheckSeconds = 10
	}
	if config.StatusCheckDown <= 0 {
		config.StatusCheckDown = 360
	}

	return config, nil
}

func loadSSHConfiguration(configSSHFile string, verbose, debug bool) (*utils.ManagerSSHConfig, error) {
	if configSSHFile == "" {
		return nil, fmt.Errorf("no config SSH file configured")
	}

	configSSH, err := loadManagerSSHConfig(configSSHFile, verbose, debug)
	if err != nil {
		return nil, fmt.Errorf("Error loading config SSH file")
	}
	return configSSH, nil
}

func loadCloudConfiguration(configCloudFile string, verbose, debug bool) (*utils.ManagerCloudConfig, error) {
	if configCloudFile == "" {
		return nil, fmt.Errorf("no config Cloud file configured")
	}

	configCloud, err := loadManagerCloudConfig(configCloudFile, verbose, debug)
	if err != nil {
		return nil, fmt.Errorf("Error loading config Cloud file")

	}

	return configCloud, nil
}

func processCloudConfiguration(configCloud *utils.ManagerCloudConfig, configSSH *utils.ManagerSSHConfig, verbose, debug bool) error {
	switch configCloud.Provider {
	case "digitalocean":
		go cloud.ProcessDigitalOcean(configCloud, configSSH, verbose, debug)
	default:
		log.Fatal("Error: Unsupported cloud provider")
	}
	return nil
}

func connectToDatabase(config *utils.ManagerConfig, debug bool) *sql.DB {
	var db *sql.DB
	var err error

	for {
		if debug {
			log.Println("Manager Trying to connect to DB")
		}
		db, err = database.ConnectDB(config.DBUsername, config.DBPassword, config.DBHost, config.DBPort, config.DBDatabase, false, debug)
		if err != nil {
			log.Printf("Error connecting to DB: %v", err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
	return db
}

func setInitialTaskStatus(db *sql.DB, verbose, debug bool) {
	if debug {
		log.Println("Manager Setting tasks with running status to failed")
	}
	var wg sync.WaitGroup
	if err := database.SetTasksStatusIfStatus("running", db, "failed", verbose, debug, &wg); err != nil {
		log.Printf("Error setting task statuses: %v", err)
	}
}

func initializeHTTPClient(config *utils.ManagerConfig, verifyAltName, verbose, debug bool) {
	var err error
	if config.CertFolder != "" {
		config.ClientHTTP, err = utils.CreateTLSClientWithCACert(config.CertFolder+"/ca-cert.pem", verifyAltName, verbose, debug)
		if err != nil {
			log.Printf("Error creating HTTP client: %v", err)
			return
		}
	} else {
		config.ClientHTTP = &http.Client{}
	}
	config.ClientHTTP.Timeout = 5 * time.Second
}

func startSSHBackgroundTask(configSSH *utils.ManagerSSHConfig, config *utils.ManagerConfig, verbose, debug bool) {
	go sshtunnel.StartSSH(configSSH, config.HTTPPort, config.HTTPSPort, verbose, debug)
}
func startBackgroundTask(db *sql.DB, config *utils.ManagerConfig, wg *sync.WaitGroup, writeLock *sync.Mutex, verbose, debug bool) {
	go utils.VerifyWorkersLoop(db, config, verbose, debug, wg, writeLock)
	go utils.ManageTasks(config, db, verbose, debug, wg, writeLock)
}

func setupAndStartServers(swagger bool, config *utils.ManagerConfig, db *sql.DB, wg *sync.WaitGroup, writeLock *sync.Mutex, verbose, debug bool) {
	router := mux.NewRouter()
	amw := authenticationMiddleware{
		tokenUsers:   make(map[string]string),
		tokenWorkers: make(map[string]string),
	}
	amw.Populate(config)

	if swagger {
		startSwaggerWeb(router, verbose, debug)
	}

	// Set up routes
	setupRoutes(router, config, db, verbose, debug, wg, writeLock, amw)

	// Start servers
	startServers(router, config, verbose, debug)
}

func setupRoutes(router *mux.Router, config *utils.ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup, writeLock *sync.Mutex, amw authenticationMiddleware) {
	status := router.PathPrefix("/status").Subrouter()
	status.Use(amw.Middleware)
	status.HandleFunc("", func(w http.ResponseWriter, r *http.Request) {
		api.HandleStatus(w, r, db, verbose, debug)
	}).Methods("GET")

	workers := router.PathPrefix("/worker").Subrouter()
	workers.Use(amw.Middleware)
	addHandleWorker(workers, config, db, verbose, debug, wg, writeLock)

	task := router.PathPrefix("/task").Subrouter()
	task.Use(amw.Middleware)
	addHandleTask(task, config, db, verbose, debug, wg, writeLock)

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "Apache")
			next.ServeHTTP(w, r)
		})
	})
}

func startServers(router *mux.Router, config *utils.ManagerConfig, verbose, debug bool) {
	var wgServer sync.WaitGroup

	if config.CertFolder != "" && config.HTTPSPort > 0 {
		httpsAddr := fmt.Sprintf(":%d", config.HTTPSPort)
		if verbose || debug {
			log.Printf("Starting HTTPS server on port %d", config.HTTPSPort)
		}
		httpsServer := &http.Server{
			Addr:         httpsAddr,
			Handler:      router,
			ReadTimeout:  time.Duration(config.APIReadTimeout) * time.Second,
			WriteTimeout: time.Duration(config.APIWriteTimeout) * time.Second,
			IdleTimeout:  time.Duration(config.APIIdleTimeout) * time.Second,
		}
		go func() {
			if err := httpsServer.ListenAndServeTLS(config.CertFolder+"/cert.pem", config.CertFolder+"/key.pem"); err != nil {
				log.Fatalf("Error starting HTTPS server: %v", err)
			}
		}()
		wgServer.Add(1)
	}

	if config.HTTPPort > 0 {
		httpAddr := fmt.Sprintf(":%d", config.HTTPPort)
		if verbose || debug {
			log.Printf("Starting HTTP server on port %d", config.HTTPPort)
		}
		httpServer := &http.Server{
			Addr:         httpAddr,
			Handler:      router,
			ReadTimeout:  time.Duration(config.APIReadTimeout) * time.Second,
			WriteTimeout: time.Duration(config.APIWriteTimeout) * time.Second,
			IdleTimeout:  time.Duration(config.APIIdleTimeout) * time.Second,
		}
		go func() {
			if err := httpServer.ListenAndServe(); err != nil {
				log.Fatalf("Error starting HTTP server: %v", err)
			}
		}()
		wgServer.Add(1)
	}

	wgServer.Wait()
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
				ctx := context.WithValue(r.Context(), utils.UsernameKey, user)

				// Pass down the request with the updated context to the next middleware (or final handler)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else if foundWorker {
				// We found the token in our map
				// Add the username to the request context
				ctx := context.WithValue(r.Context(), utils.WorkerKey, worker)

				// Pass down the request with the updated context to the next middleware (or final handler)
				next.ServeHTTP(w, r.WithContext(ctx))

			} else {
				// Write an error and stop the handler chain
				http.Error(w, "{ \"error\" : \"Forbidden\" }", http.StatusForbidden)

			}
		}
	})
}
