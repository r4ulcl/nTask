package utils

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/r4ulcl/nTask/manager/database"
)

func ManageTasks(config *ManagerConfig, db *sql.DB, verbose, debug bool, wg *sync.WaitGroup) {
	// infinite loop eecuted with go routine
	for {
		// Get all tasks in order and if priority
		tasks, err := database.GetTasksPending(100, db, verbose, debug)
		if err != nil {
			log.Println(err.Error())
		}

		// Get iddle workers
		workers, err := database.GetWorkerIddle(db, verbose, debug)
		if err != nil {
			log.Println(err.Error())
		}

		if debug {
			log.Println("tasks", len(tasks))
			log.Println("workers", len(workers))
		}

		// if there are tasks
		if len(tasks) > 0 && len(workers) > 0 {
			for _, task := range tasks {
				for _, worker := range workers {
					// if WorkerName not send or set this worker, just sendAddTask
					if task.WorkerName == "" || task.WorkerName == worker.Name {
						err = SendAddTask(db, config, &worker, &task, verbose, debug, wg)
						if err != nil {
							log.Println("Error SendAddTask", err.Error())
							//time.Sleep(time.Second * 1)
							break
						}
					}
				}
				// Update iddle workers after loop all
				workers, err = database.GetWorkerIddle(db, verbose, debug)
				if err != nil {
					log.Println(err.Error())
				}
				// If no workers just start again
				if len(workers) == 0 {
					break
				}
			}
		} else {
			time.Sleep(time.Second * 1)
		}
	}
}
