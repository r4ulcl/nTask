package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	globalStructs "github.com/r4ulcl/NetTask/globalStructs"
)

func AddTask(db *sql.DB, task globalStructs.Task) error {
	// Insert the JSON data into the MySQL table
	argsString := strings.Join(task.Args, ",")
	_, err := db.Exec("INSERT INTO task (ID, module, args, status, WorkerName, output) VALUES (?, ?, ?, ?, ?, ?)",
		task.ID, task.Module, argsString, task.Status, task.WorkerName, task.Output)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalStructs.Task) error {
	// Update all fields in the MySQL table
	argsString := strings.Join(task.Args, ",")
	_, err := db.Exec("UPDATE task SET module=?, args=?, status=?, WorkerName=?, output=? WHERE ID=?",
		task.Module, argsString, task.Status, task.WorkerName, task.Output, task.ID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func RmTask(db *sql.DB, ID string) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM task WHERE ID = ?"
	fmt.Println("ID: ", ID)
	result, err := db.Exec(sqlStatement, ID)
	if err != nil {
		return err
	}

	a, _ := result.RowsAffected()

	if a < 1 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func GetTasks(db *sql.DB) ([]globalStructs.Task, error) {
	sql := "SELECT ID, module, args, created_at, updated_at, status, WorkerName, output, priority FROM task ORDER BY priority DESC, created_at ASC"
	return GetTasksSQL(sql, db)
}

func GetTasksPending(db *sql.DB) ([]globalStructs.Task, error) {
	sql := "SELECT ID, module, args, created_at, updated_at, status, WorkerName, output, priority FROM task WHERE status = 'pending' ORDER BY priority DESC, created_at ASC"
	return GetTasksSQL(sql, db)
}

func GetTasksSQL(sql string, db *sql.DB) ([]globalStructs.Task, error) {
	var tasks []globalStructs.Task

	// Query all tasks from the task table
	rows, err := db.Query(sql)
	if err != nil {
		fmt.Println(err)
		return tasks, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var ID string
		var module string
		var args string
		var created_at string
		var updated_at string
		var status string
		var WorkerName string
		var output string
		var priority bool

		// Scan the values from the row into variables
		err := rows.Scan(&ID, &module, &args, &created_at, &updated_at, &status, &WorkerName, &output, &priority)
		if err != nil {
			fmt.Println(err)
			return tasks, err
		}

		// Data into a Person struct
		var task globalStructs.Task
		task.ID = ID
		task.Module = module
		task.Args = strings.Split(args, ",")
		task.Created_at = created_at
		task.Updated_at = updated_at
		task.Status = status
		task.WorkerName = WorkerName
		task.Output = output
		task.Priority = priority

		// Append the person to the slice
		tasks = append(tasks, task)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		fmt.Println(err)
		return tasks, err
	}

	return tasks, nil
}

func GetTask(db *sql.DB, ID string) (globalStructs.Task, error) {
	var task globalStructs.Task
	// Retrieve the JSON data from the MySQL table
	var module string
	var args string
	var created_at string
	var updated_at string
	var status string
	var WorkerName string
	var output string
	err := db.QueryRow("SELECT ID, created_at, updated_at, module, args, status, WorkerName, output FROM task WHERE ID = ?",
		ID).Scan(&ID, &created_at, &updated_at, &module, &args, &status, &WorkerName, &output)
	if err != nil {
		log.Println(err)
		return task, err
	}

	// Data back to a struct
	task.ID = ID
	task.Module = module
	task.Args = strings.Split(args, ",")
	task.Created_at = created_at
	task.Updated_at = updated_at
	task.Status = status
	task.WorkerName = WorkerName
	task.Output = output

	return task, nil
}

func GetTaskWorker(db *sql.DB, ID string) (string, error) {
	// Retrieve the JSON data from the MySQL table
	var workerName string
	err := db.QueryRow("SELECT WorkerName FROM task WHERE ID = ?",
		ID).Scan(&workerName)
	if err != nil {
		log.Println(err)
		return workerName, err
	}

	return workerName, nil
}

// SetTaskOutput save the output of the task in the DB
func SetTaskOutput(db *sql.DB, ID, output string) error {
	_, err := db.Exec("UPDATE task SET output = ? WHERE ID = ?",
		output, ID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetTaskWorkerName save the output of the task in the DB
func SetTaskWorkerName(db *sql.DB, ID, workerName string) error {
	_, err := db.Exec("UPDATE task SET workerName = ? WHERE ID = ?",
		workerName, ID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetTaskStatus save the output of the task in the DB
func SetTaskStatus(db *sql.DB, ID, status string) error {
	_, err := db.Exec("UPDATE task SET status = ? WHERE ID = ?",
		status, ID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
