// workerouter.go
package worker

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/managerRequest"
	"github.com/r4ulcl/nTask/worker/utils"
	"github.com/r4ulcl/nTask/worker/websockets"
)

func StartWorker(swagger bool, configFile string, verifyAltName, verbose, debug bool) {
	log.Println("Running as worker router...")

	config, err := utils.LoadWorkerConfig(configFile, verbose, debug)
	if err != nil {
		log.Fatal("Error loading config file: ", err)
	}

	var writeLock sync.Mutex

	status := globalstructs.WorkerStatus{
		Name:         config.Name,
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
		err := managerRequest.DeleteWorker(config, verbose, debug, &writeLock)
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
		conn, err := managerRequest.CreateWebsocket(config, verbose, debug)
		if err != nil {
			if verbose {
				log.Println("Error worker: ", err)
			}
		} else {
			config.Conn = conn

			err = managerRequest.AddWorker(config, verbose, debug, &writeLock)
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
		}
		time.Sleep(time.Second * 5)
	}

	go websockets.GetMessage(config, &status, verbose, debug, &writeLock)

	go websockets.RecreateConnection(config, verbose, debug, &writeLock)

	mainloop()
}

func mainloop() {
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
