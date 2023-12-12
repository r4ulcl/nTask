// workerouter.go
package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/api"
	"github.com/r4ulcl/nTask/worker/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

func loadWorkerConfig(filename string, verbose bool) (*utils.WorkerConfig, error) {
	var config utils.WorkerConfig
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		if verbose {
			log.Println("Error reading worker config file: ", err)
		}
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		if verbose {
			log.Println("Error unmarshalling worker config: ", err)
		}
		return &config, err
	}

	// if Name is empty use hostname
	if config.Name == "" {
		hostname := ""
		hostname, err = os.Hostname()
		if err != nil {
			if verbose {
				log.Println("Error getting hostname:", err)
			}
			return &config, err
		}
		config.Name = hostname
	}

	// if OauthToken is empty create a new token
	if config.OAuthToken == "" {
		config.OAuthToken, err = utils.GenerateToken(32, verbose)
		if err != nil {
			if verbose {
				log.Println("Error generating OAuthToken:", err)
			}
			return &config, err
		}
		fmt.Println(config.OAuthToken)
	}

	// Print the values from the struct
	if verbose {
		log.Println("Name:", config.Name)
		log.Println("Tasks:")

		for module, exec := range config.Modules {
			log.Printf("  Module: %s, Exec: %s\n", module, exec)
		}
	}

	return &config, nil
}

func checkIPMiddleware(allowedIP string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			if clientIP != allowedIP {
				// Optionally, log or handle unauthorized access here
				return // Do not respond, just exit the middleware
			}
			next.ServeHTTP(w, r)
		})
	}
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

func StartWorker(swagger bool, configFile, certFolder string, verifyAltName, verbose bool) {
	log.Println("Running as worker router...")

	// if config file empty set default
	if configFile == "" {
		configFile = "worker.conf"
	}

	config, err := loadWorkerConfig(configFile, verbose)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	status := globalstructs.WorkerStatus{
		IddleThreads: config.IddleThreads,
		WorkingIDs:   make(map[string]int),
	}

	// Create a channel to receive signals for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	// Notify the sigChan for interrupt signals (e.g., Ctrl+C)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	// Create a goroutine to handle the signal
	go func(config *utils.WorkerConfig) {
		// Wait for a signal
		sig := <-sigChan
		fmt.Println("\nReceived signal:", sig)

		// Execute your function or cleanup here
		fmt.Println("Executing cleanup function...")
		// Your function code here
		err := utils.DeleteWorker(config, verbose)
		if err != nil {
			log.Println(err)
		}

		// Exit the program gracefully
		os.Exit(0)
	}(config)

	if certFolder != "" {
		// Create an HTTP client with the custom TLS configuration
		clientHTTP, err := utils.CreateTLSClientWithCACert(certFolder+"/ca-cert.pem", verifyAltName, verbose)
		if err != nil {
			fmt.Println("Error creating HTTP client:", err)
			return
		}

		config.ClientHTTP = clientHTTP
	} else {
		config.ClientHTTP = &http.Client{}
	}
	// Loop until connects
	for {
		err = utils.AddWorker(config, verbose)
		if err != nil {
			if verbose {
				log.Println(err)
			}
		} else {
			break
		}
		time.Sleep(time.Second * 5)
	}

	router := mux.NewRouter()

	// Only allow API from manager
	router.Use(checkIPMiddleware(config.ManagerIP))

	if swagger {
		// Start swagger endpoint
		startSwaggerWeb(router, verbose)
	}

	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetStatus(w, r, &status, config, verbose)
	}).Methods("GET") // check worker status

	// Task
	router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, &status, config, verbose)
	}).Methods("POST") // Add task

	router.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, &status, config, verbose)
	}).Methods("DELETE") // delete task

	http.Handle("/", router)

	// Set string for the port
	addr := fmt.Sprintf(":%s", config.Port)
	if verbose {
		log.Println(addr)
	}

	// if there is cert is HTTPS
	if certFolder != "" {
		log.Fatal(http.ListenAndServeTLS(addr, certFolder+"/cert.pem", certFolder+"/key.pem", router))
	} else {
		err = http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
