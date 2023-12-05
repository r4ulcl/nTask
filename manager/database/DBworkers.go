package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

// AddWorker adds a worker to the database.
func AddWorker(db *sql.DB, worker *globalstructs.Worker) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, ip, port, oauthToken, working, up, count) VALUES (?, ?, ?, ?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, worker.OauthToken, worker.Working, worker.UP, worker.Count)
	if err != nil {
		return err
	}
	return nil
}

// RmWorkerName deletes a worker by its name.
func RmWorkerName(db *sql.DB, name string) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM worker WHERE name = ?"
	log.Println("Name: ", name)
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
func RmWorkerIPPort(db *sql.DB, ip, port string) error {
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
func GetWorkers(db *sql.DB) ([]globalstructs.Worker, error) {
	// Slice to store all workers
	var workers []globalstructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query("SELECT name, ip, port, oauthToken, working, up, count FROM worker")
	if err != nil {
		log.Println(err)
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
		var working bool
		var up bool
		var count int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &working, &up, &count)
		if err != nil {
			log.Println(err)
			return workers, err
		}

		// Data into a Worker struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.Working = working
		worker.UP = up
		worker.Count = count

		// Append the worker to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Println(err)
		return workers, err
	}

	return workers, nil
}

// GetWorker retrieves a worker from the database by its name.
func GetWorker(db *sql.DB, name string) (globalstructs.Worker, error) {
	var worker globalstructs.Worker
	// Retrieve the JSON data from the MySQL table
	var name2 string
	var ip string
	var port string
	var oauthToken string
	var working bool
	var up bool
	var count int

	err := db.QueryRow("SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE name = ?", name).Scan(&name2, &ip, &port, &oauthToken, &working, &up, &count)
	if err != nil {
		log.Println(err)
		return worker, err
	}

	// Data into the struct
	worker.Name = name
	worker.IP = ip
	worker.Port = port
	worker.OauthToken = oauthToken
	worker.Working = working
	worker.UP = up
	worker.Count = count

	return worker, nil
}

// UpdateWorker updates the information of a worker in the database.
func UpdateWorker(db *sql.DB, worker *globalstructs.Worker) error {
	// Update the JSON data in the MySQL table based on the worker's name
	_, err := db.Exec("UPDATE worker SET name = ?, ip = ?, port = ?, oauthToken = ?,working = ?, UP = ? WHERE name = ?",
		worker.IP, worker.Port, worker.OauthToken, worker.Working, worker.UP, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// SetWorkerUPto sets the status of a worker to the specified value.
func SetWorkerUPto(up bool, db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET UP = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// SetWorkerworkingTo sets the status of a worker to the specified working value.
func SetWorkerworkingTo(working bool, db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// SetWorkerworkingToString sets the status of a worker to the specified working value using the worker's name.
func SetWorkerworkingToString(working bool, db *sql.DB, worker string) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// GetWorkerIddle retrieves all workers that are iddle.
func GetWorkerIddle(db *sql.DB) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true AND working = false;"
	return GetWorkerSQL(sql, db)
}

// GetWorkerUP retrieves all workers that are up.
func GetWorkerUP(db *sql.DB) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true;"
	return GetWorkerSQL(sql, db)
}

// GetWorkerSQL retrieves workers information based on a SQL statement.
func GetWorkerSQL(sql string, db *sql.DB) ([]globalstructs.Worker, error) {
	// Slice to store all workers
	var workers []globalstructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
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
		var working bool
		var up bool
		var count int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &working, &up, &count)
		if err != nil {
			log.Println(err)
			return workers, err
		}

		// Data into a Worker struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.Working = working
		worker.UP = up
		worker.Count = count

		// Append the worker to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Println(err)
		return workers, err
	}

	return workers, nil
}

// GetWorkerCount get workers count by name (used to count until 3 to set down)
func GetWorkerCount(db *sql.DB, worker *globalstructs.Worker) (int, error) {
	var countS string
	err := db.QueryRow("SELECT count FROM worker WHERE name = ?",
		worker.Name).Scan(&countS)
	if err != nil {
		log.Println(err)
		return -1, err
	}
	count, err := strconv.Atoi(countS)
	if err != nil {
		return -1, err
	}

	log.Println("count", count)
	return count, nil
}

// SetWorkerCount set worker count to count int
func SetWorkerCount(count int, db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET count = ? WHERE name = ?",
		count, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// AddWorkerCount add 1 to worker count
func AddWorkerCount(db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET count = count + 1 WHERE name = ?",
		worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
