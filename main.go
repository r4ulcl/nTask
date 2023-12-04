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
	flag.BoolVar(&isManager, "manager", false, "Run as manager (default is worker)")
	flag.BoolVar(&isWorker, "worker", false, "Run as worker (default is api client)")

	flag.Parse()

	// Check the argument and call the appropriate function
	switch {
	case isManager:
		manager.StartManager()
	case isWorker:
		worker.StartWorker()
	default:
		log.Println("TODO, use api from cmd")
	}
}
