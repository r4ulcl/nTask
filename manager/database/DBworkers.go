package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
)

func AddWorker(db *sql.DB, worker *globalStructs.Worker) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO worker (name, ip, port, oauthToken, working, up, count) VALUES (?, ?, ?, ?, ?, ?, ?)",
		worker.Name, worker.IP, worker.Port, worker.OauthToken, worker.Working, worker.UP, worker.Count)
	if err != nil {
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

func GetWorkers(db *sql.DB) ([]globalStructs.Worker, error) {
	// Slice to store all workers
	var workers []globalStructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query("SELECT name, ip, port, oauthToken, working, up, count FROM worker")
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
		var oauthToken string
		var working bool
		var up bool
		var count int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &working, &up, &count)
		if err != nil {
			fmt.Println(err)
			return workers, err
		}

		// Data into a Person struct
		var worker globalStructs.Worker
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
		fmt.Println(err)
		return workers, err
	}

	return workers, nil
}

func GetWorker(db *sql.DB, name string) (globalStructs.Worker, error) {
	var worker globalStructs.Worker
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

func UpdateWorker(db *sql.DB, worker *globalStructs.Worker) error {
	// Update the JSON data in the MySQL table based on the worker's name
	_, err := db.Exec("UPDATE worker SET name = ?, ip = ?, port = ?, oauthToken = ?,working = ?, UP = ? WHERE name = ?",
		worker.IP, worker.Port, worker.OauthToken, worker.Working, worker.UP, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetWorkerDown set worker status to status var, false -> cant connect
func SetWorkerUPto(up bool, db *sql.DB, worker *globalStructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET UP = ? WHERE name = ?",
		up, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func SetWorkerworkingTo(working bool, db *sql.DB, worker *globalStructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func SetWorkerworkingToString(working bool, db *sql.DB, worker string) error {
	_, err := db.Exec("UPDATE worker SET working = ? WHERE name = ?",
		working, worker)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func GetWorkerIddle(db *sql.DB) ([]globalStructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true AND working = false;"
	return GetWorkerSQL(sql, db)
}

func GetWorkerUP(db *sql.DB) ([]globalStructs.Worker, error) {
	sql := "SELECT name, ip, port, oauthToken, working, up, count FROM worker WHERE up = true;"
	return GetWorkerSQL(sql, db)
}

func GetWorkerSQL(sql string, db *sql.DB) ([]globalStructs.Worker, error) {

	// Slice to store all workers
	var workers []globalStructs.Worker

	// Query all workers from the worker table
	rows, err := db.Query(sql)
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
		var oauthToken string
		var working bool
		var up bool
		var count int

		// Scan the values from the row into variables
		err := rows.Scan(&name, &ip, &port, &oauthToken, &working, &up, &count)
		if err != nil {
			fmt.Println(err)
			return workers, err
		}

		// Data into a Person struct
		var worker globalStructs.Worker
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
		fmt.Println(err)
		return workers, err
	}

	return workers, nil
}

func GetWorkerCount(db *sql.DB, worker *globalStructs.Worker) (int, error) {
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

	fmt.Println("count", count)
	return count, nil
}

func SetWorkerCount(count int, db *sql.DB, worker *globalStructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET count = ? WHERE name = ?",
		count, worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func AddWorkerCount(db *sql.DB, worker *globalStructs.Worker) error {
	_, err := db.Exec("UPDATE worker SET count = count + 1 WHERE name = ?",
		worker.Name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
