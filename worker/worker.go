// Package worker for all the workers data
package worker

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/managerrequest"
	"github.com/r4ulcl/nTask/worker/utils"
	"github.com/r4ulcl/nTask/worker/websockets"
)

func StartWorker(swagger bool, configFile string, verifyAltName, verbose, debug bool) {
	log.Println("Worker Running as worker router...")

	config, err := utils.LoadWorkerConfig(configFile, verbose, debug)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	var writeLock sync.Mutex

	status := globalstructs.WorkerStatus{
		Name:         config.Name,
		IddleThreads: config.DefaultThreads,
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
		log.Println("\nReceived signal:", sig)

		// Execute your function or cleanup here
		log.Println("Executing cleanup function...")

		//delete worker
		if config.Conn != nil {
			err := managerrequest.DeleteWorker(config, verbose, debug, &writeLock)
			if err != nil {
				log.Println("Worker Error worker DeleteWorker: ", err)
			}
		}
		// Exit the program gracefully
		os.Exit(0)
	}(config)

	if config.CA != "" {
		// Create an HTTP client with the custom TLS configuration
		clientHTTP, err := utils.CreateTLSClientWithCACert(config.CA, verifyAltName, verbose, debug)
		if err != nil {
			log.Println("Error creating HTTPS client:", err)
			return
		}

		config.ClientHTTP = clientHTTP
	} else {
		config.ClientHTTP = &http.Client{}
	}

	websockets.CreateConnection(config, verifyAltName, verbose, debug, &writeLock)

	go websockets.GetMessage(config, &status, verbose, debug, &writeLock)

	go websockets.RecreateConnection(config, verifyAltName, verbose, debug, &writeLock)

	mainloop()
}

func mainloop() {
	exitSignal := make(chan os.Signal, 1) // Use a buffered channel with capacity 1
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
