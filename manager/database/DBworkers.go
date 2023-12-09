package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

// AddWorker adds a worker to the database.
func AddWorker(db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, ip, port, oauthToken, IddleThreads, up, downCount)"+
		" VALUES (?, ?, ?, ?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, worker.OauthToken, worker.IddleThreads, worker.UP, worker.DownCount)
	if err != nil {
		return err
	}
	return nil
}

// RmWorkerName deletes a worker by its name.
func RmWorkerName(db *sql.DB, name string, verbose bool) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM worker WHERE name = ?"
	log.Println("Delete worker Name: ", name)
	result, err := db.Exec(sqlStatement, name)
	if err != nil {
		return err
	}

	a, _ := result.RowsAffected()

	if a < 1 {
		return fmt.Errorf("worker not found")
	}

	return nil
}

// RmWorkerIPPort deletes a worker by its IP and Port.
func RmWorkerIPPort(db *sql.DB, ip, port string, verbose bool) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM worker WHERE IP = ? AND port = ?"
	result, err := db.Exec(sqlStatement, ip, port)
	if err != nil {
		return err
	}

	a, _ := result.RowsAffected()

	if a < 1 {
		return fmt.Errorf("worker not found")
	}

	return nil
}

// GetWorkers retrieves all workers from the database.
func GetWorkers(db *sql.DB, verbose bool) ([]globalstructs.Worker, error) {
	// Slice to store all workers
	var workers []globalstructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query("SELECT name, ip, port, oauthToken, IddleThreads,  up, downCount FROM worker")
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return workers, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var name string
		var ip string
		var port string
		var oauthToken string
		var IddleThreads int
		var up bool
		var downCount int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &IddleThreads, &up, &downCount)
		if err != nil {
			if verbose {
				log.Println(err)
			}
			return workers, err
		}

		// Data into a Worker struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.IddleThreads = IddleThreads
		worker.UP = up
		worker.DownCount = downCount

		// Append the worker to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		if verbose {
			log.Println(err)
		}
		return workers, err
	}

	return workers, nil
}

// GetWorker retrieves a worker from the database by its name.
func GetWorker(db *sql.DB, name string, verbose bool) (globalstructs.Worker, error) {
	var worker globalstructs.Worker
	// Retrieve the JSON data from the MySQL table
	var name2 string
	var ip string
	var port string
	var oauthToken string
	var IddleThreads int
	var up bool
	var downCount int

	err := db.QueryRow("SELECT name, ip, port, oauthToken, IddleThreads,  up, downCount FROM worker WHERE name = ?",
		name).Scan(
		&name2, &ip, &port, &oauthToken, &IddleThreads, &up, &downCount)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return worker, err
	}

	// Data into the struct
	worker.Name = name
	worker.IP = ip
	worker.Port = port
	worker.OauthToken = oauthToken
	worker.IddleThreads = IddleThreads
	worker.UP = up
	worker.DownCount = downCount

	return worker, nil
}

// UpdateWorker updates the information of a worker in the database.
func UpdateWorker(db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	// Update the JSON data in the MySQL table based on the worker's name
	_, err := db.Exec("UPDATE worker SET name = ?, ip = ?, port = ?, oauthToken = ?,"+
		" IddleThreads = ?, UP = ?, downCount = ? WHERE name = ?",
		worker.IP, worker.Port, worker.OauthToken, worker.IddleThreads, worker.UP, worker.DownCount, worker.Name)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// SetWorkerOauthToken sets oauth token to new value.
func SetWorkerOauthToken(oauthToken string, db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET oauthToken = ? WHERE name = ?",
		oauthToken, worker.Name)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}

	return nil
}

// SetWorkerUPto sets the status of a worker to the specified value.
func SetWorkerUPto(up bool, db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET UP = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}

	return nil
}

// SetWorkerworkingToString sets the status of a worker to the specified working value using the worker's name.
func SetWorkerworkingTo(IddleThreads int, db *sql.DB, worker string, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET IddleThreads = ? WHERE name = ?",
		IddleThreads, worker)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}

	return nil
}

// SetWorkerworkingToString sets the status of a worker to the specified working value using the worker's name.
func AddWorkerIddleThreads1(db *sql.DB, worker string, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET IddleThreads = IddleThreads + 1 WHERE name = ?;",
		worker)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// SetWorkerworkingToString sets the status of a worker to the specified working value using the worker's name.
func SubtractWorkerIddleThreads1(db *sql.DB, worker string, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET IddleThreads = CASE WHEN IddleThreads > 0 THEN IddleThreads - 1 "+
		"ELSE 0 END WHERE name = ?", worker)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// GetWorkerIddle retrieves all workers that are iddle.
func GetWorkerIddle(db *sql.DB, verbose bool) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, IddleThreads, up, downCount FROM worker WHERE up = true AND IddleThreads > 0;"
	return GetWorkerSQL(sql, db, verbose)
}

// GetWorkerUP retrieves all workers that are up.
func GetWorkerUP(db *sql.DB, verbose bool) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, IddleThreads, up, downCount FROM worker WHERE up = true;"
	return GetWorkerSQL(sql, db, verbose)
}

// GetWorkerSQL retrieves workers information based on a SQL statement.
func GetWorkerSQL(sql string, db *sql.DB, verbose bool) ([]globalstructs.Worker, error) {
	// Slice to store all workers
	var workers []globalstructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query(sql)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return workers, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var name string
		var ip string
		var port string
		var oauthToken string
		var IddleThreads int
		var up bool
		var downCount int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &IddleThreads, &up, &downCount)
		if err != nil {
			if verbose {
				log.Println(err)
			}
			return workers, err
		}

		// Data into a Worker struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.IddleThreads = IddleThreads
		worker.UP = up
		worker.DownCount = downCount

		// Append the worker to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		if verbose {
			log.Println(err)
		}
		return workers, err
	}

	return workers, nil
}

// GetWorkerCount get workers downCount by name (used to downCount until 3 to set down)
func GetWorkerDownCount(db *sql.DB, worker *globalstructs.Worker, verbose bool) (int, error) {
	var countS string
	err := db.QueryRow("SELECT downCount FROM worker WHERE name = ?",
		worker.Name).Scan(&countS)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return -1, err
	}
	downCount, err := strconv.Atoi(countS)
	if err != nil {
		return -1, err
	}

	log.Println("count", downCount)
	return downCount, nil
}

// SetWorkerCount set worker downCount to downCount int
func SetWorkerDownCount(count int, db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET downCount = ? WHERE name = ?",
		count, worker.Name)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}

	return nil
}

// AddWorkerCount add 1 to worker downCount
func AddWorkerDownCount(db *sql.DB, worker *globalstructs.Worker, verbose bool) error {
	_, err := db.Exec("UPDATE worker SET downCount = downCount + 1 WHERE name = ?",
		worker.Name)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}

	return nil
}
