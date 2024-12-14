package utils

import (
	"database/sql"
	"sync"

	"github.com/r4ulcl/nTask/globalstructs"
	"github.com/r4ulcl/nTask/manager/database"

	"github.com/go-sql-driver/mysql"
)

// HandleAddWorkerError func to handle error adding workers
func HandleAddWorkerError(err error, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		// Handle the MySQL duplicate entry error
		if mysqlErr.Number == 1062 { // MySQL error number for duplicate entry
			// Set all worker tasks to 'pending' with REDO status
			err = database.SetTasksWorkerPending(db, worker.Name, verbose, debug, wg)
			if err != nil {
				return err
			}

			// Update worker record
			err = database.UpdateWorker(db, worker, verbose, debug, wg)
			if err != nil {
				return err
			}

			// Reset the worker's down count
			err = database.SetWorkerDownCount(0, db, worker, verbose, debug, wg)
			if err != nil {
				return err
			}
		}
		// Return the original error if it's not the duplicate entry error
		return err
	}
	// Return nil if no error
	return nil
}
