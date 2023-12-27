// workerouter.go
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/api"
	"github.com/r4ulcl/nTask/worker/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

func loadWorkerConfig(filename string, verbose, debug bool) (*utils.WorkerConfig, error) {
	var config utils.WorkerConfig
	content, err := os.ReadFile(filename)
	if err != nil {
		if debug {
			log.Println("Error reading worker config file: ", err)
		}
		return &config, err
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		if debug {
			log.Println("Error unmarshalling worker config: ", err)
		}
		return &config, err
	}

	// if Name is empty use hostname
	if config.Name == "" {
		hostname := ""
		hostname, err = os.Hostname()
		if err != nil {
			if debug {
				log.Println("Error getting hostname:", err)
			}
			return &config, err
		}
		config.Name = hostname
	}

	// if OauthToken is empty create a new token
	if config.OAuthToken == "" {
		config.OAuthToken, err = utils.GenerateToken(32, verbose, debug)
		if err != nil {
			if debug {
				log.Println("Error generating OAuthToken:", err)
			}
			return &config, err
		}
		fmt.Println(config.OAuthToken)
	}

	// Print the values from the struct
	if debug {
		log.Println("Name:", config.Name)
		log.Println("Tasks:")

		for module, exec := range config.Modules {
			log.Printf("  Module: %s, Exec: %s\n", module, exec)
		}
	}

	return &config, nil
}

func getDockerDomain(internalIP string) (string, error) {
	addrs, err := net.LookupAddr(internalIP)
	if err != nil {
		return "", err
	}

	// The returned address might be in the form "hostname.domain".
	// We want to extract the Docker service name, which is the part before the first dot.
	parts := strings.Split(addrs[0], ".")
	dockerDomain := parts[0]

	return dockerDomain, nil
}

func checkIPMiddleware(allowedIP string, verbose, debug bool) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			if clientIP != allowedIP {
				container_name, _ := getDockerDomain(clientIP)
				if container_name != allowedIP {
					// Optionally, log or handle unauthorized access here
					if debug {
						log.Println("Manager IP not in whitelist, clientIP:", clientIP)
					}
					w.WriteHeader(http.StatusForbidden)
					return // Do not respond, just exit the middleware
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func startSwaggerWeb(router *mux.Router, verbose, debug bool) {
	// Serve Swagger UI at /swagger
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.json"), // URL to the swagger.json file
	))

	// Serve Swagger JSON at /swagger/doc.json
	router.HandleFunc("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/swagger.json")
	}).Methods("GET")
}

func StartWorker(swagger bool, configFile string, verifyAltName, verbose, debug bool) {
	log.Println("Running as worker router...")

	// if config file empty set default
	if configFile == "" {
		configFile = "worker.conf"
	}

	config, err := loadWorkerConfig(configFile, verbose, debug)
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

		//delete worker
		err := utils.DeleteWorker(config, verbose, debug)
		if err != nil {
			log.Println("Error worker: ", err)
		}
		// Exit the program gracefully
		os.Exit(0)
	}(config)

	if config.CertFolder != "" {
		// Create an HTTP client with the custom TLS configuration
		clientHTTP, err := utils.CreateTLSClientWithCACert(config.CertFolder+"/ca-cert.pem", verifyAltName, verbose, debug)
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
		err = utils.AddWorker(config, verbose, debug)
		if err != nil {
			if verbose {
				log.Println("Error worker: ", err)
			}
		} else {
			if verbose {
				log.Println("Worker connected to manager. ")
			}
			break
		}
		time.Sleep(time.Second * 5)
	}

	router := mux.NewRouter()

	// Only allow API from manager
	router.Use(checkIPMiddleware(config.ManagerIP, verbose, debug))

	if swagger {
		// Start swagger endpoint
		startSwaggerWeb(router, verbose, debug)
	}

	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		api.HandleGetStatus(w, r, &status, config, verbose, debug)
	}).Methods("GET") // check worker status

	// Task
	router.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskPost(w, r, &status, config, verbose, debug)
	}).Methods("POST") // Add task

	router.HandleFunc("/task/{ID}", func(w http.ResponseWriter, r *http.Request) {
		api.HandleTaskDelete(w, r, &status, config, verbose, debug)
	}).Methods("DELETE") // delete task

	http.Handle("/", router)

	// Set string for the port
	addr := fmt.Sprintf(":%s", config.Port)
	if debug {
		log.Println(addr)
	}

	// if there is cert is HTTPS
	if config.CertFolder != "" {
		log.Fatal(http.ListenAndServeTLS(addr, config.CertFolder+"/cert.pem", config.CertFolder+"/key.pem", router))
	} else {
		err = http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}
