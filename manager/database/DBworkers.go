package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/r4ulcl/NetTask/manager/utils"
)

func AddWorker(db *sql.DB, worker utils.Worker) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, ip, port, working, up) VALUES (?, ?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, worker.Working, worker.UP)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func RmWorkerName(db *sql.DB, name string) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM worker WHERE name = ?"
	fmt.Println("Name: ", name)
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

func GetWorkers(db *sql.DB) ([]utils.Worker, error) {
	// Slice to store all workers
	var workers []utils.Worker

	// Query all workers from the worker table
	rows, err := db.Query("SELECT name, ip, port, working, up FROM worker")
	if err != nil {
		fmt.Println(err)
		return workers, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var name string
		var ip string
		var port string
		var working bool
		var up bool

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &working, &up)
		if err != nil {
			fmt.Println(err)
			return workers, err
		}

		// Data into a Person struct
		var worker utils.Worker
		worker.Name = name
		worker.IP = ip
		worker.Port = port
		worker.Working = working
		worker.UP = up

		// Append the person to the slice
		workers = append(workers, worker)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		fmt.Println(err)
		return workers, err
	}

	return workers, nil
}

func GetWorker(db *sql.DB, name string) (utils.Worker, error) {
	var worker utils.Worker
	// Retrieve the JSON data from the MySQL table
	var name2 string
	var ip string
	var port string
	var working bool
	var up bool
	err := db.QueryRow("SELECT name, ip, port, working, up FROM worker WHERE name = ?", name).Scan(&name2, &ip, &port, &working, &up)
	if err != nil {
		log.Println(err)
		return worker, err
	}

	// Data back to a struct
	worker.Name = name
	worker.IP = ip
	worker.Port = port
	worker.Working = working
	worker.UP = up

	return worker, nil
}

func UpdateWorker(db *sql.DB, worker utils.Worker) error {
	// Update the JSON data in the MySQL table based on the worker's name
	_, err := db.Exec("UPDATE worker SET name = ?, ip = ?, port = ?, working = ?, UP = ? WHERE name = ?",
		worker.IP, worker.Port, worker.Working, worker.UP, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

//SetWorkerDown set worker status to status var, false -> cant connect
func SetWorkerUPto(up bool, db *sql.DB, worker utils.Worker) error {
	_, err := db.Exec("UPDATE worker SET UP = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func SetWorkerworkingTo(working bool, db *sql.DB, worker utils.Worker) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
