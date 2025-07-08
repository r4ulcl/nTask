package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// AddWorker adds a worker to the database.
func AddWorker(db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, defaultThreads, iddleThreads, up, downCount)"+
		" VALUES (?, ?, ?, ?, ?)",
		worker.Name, worker.DefaultThreads, worker.IddleThreads, worker.UP, worker.DownCount)
	if err != nil {
		return err
	}
	return nil
}

// RmWorkerName deletes a worker by its name.
func RmWorkerName(db *sql.DB, name string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM worker WHERE name = ?"
	log.Println("DB Delete worker Name: ", name)
	result, err := db.Exec(sqlStatement, name)
	if err != nil {
		return err
	}

	a, _ := result.RowsAffected()

	if a < 1 {
		return fmt.Errorf("{\"error\": \"worker not found\"}")
	}

	// Set workers task to any worker
	err = setTasksWorkerEmpty(db, name, verbose, debug, wg)
	if err != nil {
		return err
	}

	return nil
}

// GetWorkers retrieves all workers from the database.
func GetWorkers(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	sql := "SELECT name, defaultThreads, iddleThreads, up, downCount FROM worker;"
	return getWorkerSQL(sql, db, verbose, debug)
}

// GetWorker retrieves a worker from the database by its name.
func GetWorker(db *sql.DB, name string, verbose, debug bool) (globalstructs.Worker, error) {
	var worker globalstructs.Worker
	// Retrieve the JSON data from the MySQL table
	var name2 string
	var defaultThreads, iddleThreads int
	var up bool
	var downCount int

	err := db.QueryRow("SELECT name,  defaultThreads, iddleThreads, up, downCount FROM worker WHERE name = ?",
		name).Scan(
		&name2, &defaultThreads, &iddleThreads, &up, &downCount)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return worker, err
	}

	// Data into the struct
	worker.Name = name
	worker.DefaultThreads = defaultThreads
	worker.IddleThreads = iddleThreads
	worker.UP = up
	worker.DownCount = downCount

	return worker, nil
}

// UpdateWorker updates the information of a worker in the database.
func UpdateWorker(db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	// Update the JSON data in the MySQL table based on the worker's name
	_, err := db.Exec("UPDATE worker SET"+
		" defaultThreads = ?, iddleThreads = ?, up = ?, downCount = ? WHERE name = ?",
		worker.DefaultThreads, worker.IddleThreads, worker.UP, worker.DownCount, worker.Name)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}
	return nil
}

// SetWorkerUPto sets the status of a worker to the specified value.
func SetWorkerUPto(up bool, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE worker SET up = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}

	if debug {
		log.Println("DB Worker set to:", up, worker.Name)
	}

	return nil
}

// SetIddleThreadsTo sets the status of a worker to the specified iddle value using the worker's name.
func SetIddleThreadsTo(IddleThreads int, db *sql.DB, worker string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	if debug {
		log.Println("DB Set IddleThreads to", IddleThreads)
	}
	_, err := db.Exec("UPDATE worker SET IddleThreads = ? WHERE name = ?",
		IddleThreads, worker)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}

	return nil
}

// GetWorkerIddle retrieves all workers that are iddle.
func GetWorkerIddle(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	sql := "SELECT name, defaultThreads, iddleThreads, up, downCount FROM worker WHERE up = true AND IddleThreads > 0 ORDER BY RAND();"
	return getWorkerSQL(sql, db, verbose, debug)
}

// GetWorkerUP retrieves all workers that are up.
func GetWorkerUP(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	sql := "SELECT name, defaultThreads, iddleThreads, up, downCount FROM worker WHERE up = true;"
	return getWorkerSQL(sql, db, verbose, debug)
}

// getWorkerSQL retrieves workers information based on a SQL statement.
func getWorkerSQL(sql string, db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	// Slice to store all workers
	var workers []globalstructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query(sql)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return workers, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var name string
		var defaultThreads, iddleThreads int
		var up bool
		var downCount int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &defaultThreads, &iddleThreads, &up, &downCount)
		if err != nil {
			if debug {
				log.Println("DB Error DBworkers: ", err)
			}
			return workers, err
		}

		// Data into a Worker struct
		var worker globalstructs.Worker
		worker.Name = name

		worker.DefaultThreads = defaultThreads
		worker.IddleThreads = iddleThreads
		worker.UP = up
		worker.DownCount = downCount

		// Append the worker to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return workers, err
	}

	return workers, nil
}

// GetWorkerDownCount get workers downCount by name (used to downCount until 3 to set down)
func GetWorkerDownCount(db *sql.DB, worker *globalstructs.Worker, verbose, debug bool) (int, error) {
	var countS string
	err := db.QueryRow("SELECT downCount FROM worker WHERE name = ?",
		worker.Name).Scan(&countS)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return -1, err
	}
	downCount, err := strconv.Atoi(countS)
	if err != nil {
		return -1, err
	}

	if debug {
		log.Println("DB count worker:", worker.Name, "downCount:", downCount)
	}
	return downCount, nil
}

// SetWorkerDownCount set worker downCount to downCount int
func SetWorkerDownCount(count int, db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE worker SET downCount = ? WHERE name = ?",
		count, worker.Name)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}

	return nil
}

// AddWorkerDownCount add 1 to worker downCount
func AddWorkerDownCount(db *sql.DB, worker *globalstructs.Worker, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE worker SET downCount = downCount + 1 WHERE name = ?",
		worker.Name)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}

	return nil
}

// GetUpCount Get workers up count
func GetUpCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := "SELECT COUNT(*) FROM worker where up = true"

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetDownCount get count of up = false
func GetDownCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := "SELECT COUNT(*) FROM worker where up = false"

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// SubtractWorkerIddleThreads1 Subtract WorkerIddleThreads 1 if >0
func SubtractWorkerIddleThreads1(db *sql.DB, worker string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	if debug {
		log.Println("DB SubtractWorkerIddleThreads1")
	}

	_, err := db.Exec("UPDATE worker SET iddleThreads = CASE WHEN iddleThreads > 0 THEN iddleThreads - 1 "+
		"ELSE 0 END WHERE name = ?", worker)
	if err != nil {
		if debug {
			log.Println("DB Error DBworkers: ", err)
		}
		return err
	}
	return nil
}
