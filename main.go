package main

import (
	//	"os"
	"flag"

	//	"fmt"
	"github.com/r4ulcl/NetTask/manager"
	"github.com/r4ulcl/NetTask/worker"
)

func main() {

	var isManager bool
	flag.BoolVar(&isManager, "manager", false, "Run as manager (default is worker)")

	flag.Parse()

	// Check the argument and call the appropriate function
	if isManager {
		manager.StartManager()
	} else {
		worker.StartWorker()
	}
}
