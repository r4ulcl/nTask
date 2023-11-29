package main

import (
	//	"os"
	"flag"
	"fmt"

	//	"fmt"

	"github.com/r4ulcl/NetTask/manager"
	"github.com/r4ulcl/NetTask/worker"
)

// @titleNetTask API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 10.10.20.10:8180
// @BasePath /
// @Security OAuth2.0
// @SecurityDefinitions OAuth2.0

func main() {

	var isManager bool
	var isWorker bool
	flag.BoolVar(&isManager, "manager", false, "Run as manager (default is worker)")
	flag.BoolVar(&isWorker, "worker", false, "Run as worker (default is API client)")

	flag.Parse()

	// Check the argument and call the appropriate function
	if isManager {
		manager.StartManager()
	} else if isWorker {
		worker.StartWorker()
	} else {
		fmt.Println("TODO, use API from cmd")
	}
}
