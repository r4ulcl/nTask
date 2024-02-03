package process

import (
	"log"
	"sync"
	"time"

	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/worker/managerrequest"
	"github.com/r4ulcl/nTask/worker/modules"
	"github.com/r4ulcl/nTask/worker/utils"
)

var mutex sync.Mutex

// processTask is a helper function that processes the given task in the background.
// It sets the worker status to indicate that it is currently working on the task.
// It calls the ProcessModule function to execute the task's module.
// If an error occurs, it sets the task status to "failed".
// Otherwise, it sets the task status to "done" and assigns the output of the module to the task.
// Finally, it calls the CallbackTaskMessage function to send the task result to the configured callback endpoint.
// After completing the task, it resets the worker status to indicate that it is no longer working.
func Task(status *globalstructs.WorkerStatus, config *utils.WorkerConfig, task *globalstructs.Task, verbose, debug bool, writeLock *sync.Mutex) {
	//Remove one from working threads
	sustract1IddleThreads(status, verbose, debug)

	//Add one from working threads
	defer add1IddleThreads(status, verbose, debug)

	if verbose {
		log.Println("Process Start processing task", task.ID, " workCount: ", status.IddleThreads)
	}

	err := modules.ProcessModule(task, config, status, task.ID, verbose, debug)
	if err != nil {
		log.Println("Process Error ProcessModule:", err)
		task.Status = "failed"
	} else {
		task.Status = "done"
	}

	// While manager doesnt responds loop
	for {
		err = managerrequest.CallbackTaskMessage(config, task, verbose, debug, writeLock)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 10)
	}

}

func add1IddleThreads(status *globalstructs.WorkerStatus, verbose, debug bool) {
	modifyIddleThreads(status, true, verbose, debug)
}

func sustract1IddleThreads(status *globalstructs.WorkerStatus, verbose, debug bool) {
	modifyIddleThreads(status, false, verbose, debug)
}

func modifyIddleThreads(status *globalstructs.WorkerStatus, add, verbose, debug bool) {
	mutex.Lock()
	defer mutex.Unlock()

	if debug {
		log.Println("modifyIddleThreads", add)
	}

	if add {
		status.IddleThreads++
	} else {
		status.IddleThreads--
	}
}
