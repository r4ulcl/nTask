package main

import (
	// 	"os"
	"flag"
	"log"

	"github.com/r4ulcl/NetTask/manager"
	"github.com/r4ulcl/NetTask/worker"
)

// @title NetTask API
// @version 1.0
// @description NetTask API documentation
// @contact.name r4ulcl
// @contact.url https://r4ulcl.com/contact/
// @contact.email me@r4ulcl.com

// @license.name  GPL-3.0
// @license.url https://github.com/r4ulcl/NetTask/blob/main/LICENSE

// @BasePath /
// @Security OAuth2.0
// @SecurityDefinitions OAuth2.0

func main() {
	var isManager bool
	var isWorker bool
	var swagger bool
	var verbose bool
	var configFile string
	var certFile string
	var keyFile string
	flag.BoolVar(&isManager, "manager", false, "Run as manager")
	flag.BoolVar(&isWorker, "worker", false, "Run as worker")
	flag.BoolVar(&swagger, "swagger", false, "Start the swager endpoint (/swagger)")
	flag.BoolVar(&verbose, "verbose", false, "Set verbose mode")
	flag.StringVar(&configFile, "configFile", "", "Path to the config file")
	flag.StringVar(&certFile, "cert", "", "TLS certificate file cert.pem")
	flag.StringVar(&keyFile, "key", "", "TLS private key file key.pem")

	flag.Parse()

	// Check the argument and call the appropriate function
	switch {
	case isManager:
		manager.StartManager(swagger, configFile, certFile, keyFile, verbose)
	case isWorker:
		worker.StartWorker(swagger, configFile, certFile, keyFile, verbose)
	default:
		log.Println("Incorrect option")
	}
}
