package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/r4ulcl/NetTask/manager/utils"
)

func AddWorker(db *sql.DB, worker utils.Worker) error {
	// Marshal the Worker struct to JSON
	jsonData, err := json.Marshal(worker)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO worker (name, ip, port, data) VALUES (?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, jsonData)
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
	rows, err := db.Query("SELECT data FROM worker")
	if err != nil {
		fmt.Println(err)
		return workers, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var storedData []byte

		// Scan the values from the row into variables
		err := rows.Scan(&storedData)
		if err != nil {
			fmt.Println(err)
			return workers, err
		}

		// Unmarshal JSON data into a Person struct
		var worker utils.Worker
		err = json.Unmarshal(storedData, &worker)
		if err != nil {
			fmt.Println(err)
			return workers, err
		}

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
	var storedData []byte
	err := db.QueryRow("SELECT data FROM worker WHERE name = ?", name).Scan(&storedData)
	if err != nil {
		log.Println(err)
		return worker, err
	}

	// Unmarshal the JSON data back to a struct
	err = json.Unmarshal(storedData, &worker)
	if err != nil {
		log.Fatal(err)
	}

	return worker, nil
}

func UpdateWorker(db *sql.DB, worker utils.Worker) error {
	// Marshal the Worker struct to JSON
	jsonData, err := json.Marshal(worker)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Update the JSON data in the MySQL table based on the worker's name
	_, err = db.Exec("UPDATE worker SET ip = ?, port = ?, data = ? WHERE name = ?",
		worker.IP, worker.Port, jsonData, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

//SetWorkerDown set worker status to status var, false -> cant connect
func SetWorkerUPto(up bool, db *sql.DB, worker utils.Worker) error {

	//Set status to var
	worker.UP = up

	UpdateWorker(db, worker)

	return nil
}
