package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

// AddWorker add worker to DB
func AddWorker(db *sql.DB, worker *globalstructs.Worker) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, ip, port, oauthToken, working, up, count) VALUES (?, ?, ?, ?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, worker.OauthToken, worker.Working, worker.UP, worker.Count)
	if err != nil {
		return err
	}
	return nil
}

// RmWorkerName delete worker by name
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

// RmWorkerIPPort delete worker by IP and PORT
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

// GetWorkers get all workers
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

		// Data into a Person struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.Working = working
		worker.UP = up
		worker.Count = count

		// Append the person to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Println(err)
		return workers, err
	}

	return workers, nil
}

// GetWorker Get worker filter by name
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

	// Data back to a struct
	worker.Name = name
	worker.IP = ip
	worker.Port = port
	worker.OauthToken = oauthToken
	worker.Working = working
	worker.UP = up
	worker.Count = count

	return worker, nil
}

// UpdateWorker Update full worker
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

// SetWorkerDown set worker status to status var, false -> cant connect
func SetWorkerUPto(up bool, db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET UP = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// SetWorkerworkingTo set worker status to boolean working value
func SetWorkerworkingTo(working bool, db *sql.DB, worker *globalstructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// SetWorkerworkingTo set worker status to boolean working value from worker name
func SetWorkerworkingToString(working bool, db *sql.DB, worker string) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// GetWorkerIddle Get all workers iddle
func GetWorkerIddle(db *sql.DB) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true AND working = false;"
	return GetWorkerSQL(sql, db)
}

// GetWorkerUP Get all workers UP
func GetWorkerUP(db *sql.DB) ([]globalstructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true;"
	return GetWorkerSQL(sql, db)
}

// GetWorkerSQL get workers info by SQL
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

		// Data into a Person struct
		var worker globalstructs.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.OauthToken = oauthToken
		worker.Working = working
		worker.UP = up
		worker.Count = count

		// Append the person to the slice
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
