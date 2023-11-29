package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/r4ulcl/NetTask/manager/utils"
)

func AddTask(db *sql.DB, task utils.Task) error {
	// Insert the JSON data into the MySQL table
	_, err := db.Exec("INSERT INTO task (ID, status, WorkerName, output) VALUES (?, ?, ?, ?)",
		task.ID, task.Status, task.WorkerName, task.Output)
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

func StopTask(db *sql.DB, task utils.Task) error {
	fmt.Println("TODO StopTask")
	return nil
}

func GetTasks(db *sql.DB) ([]utils.Task, error) {
	var tasks []utils.Task

	// Query all tasks from the task table
	rows, err := db.Query("SELECT ID, created_at, updated_at, status, WorkerName, output FROM task")
	if err != nil {
		fmt.Println(err)
		return tasks, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var ID string
		var created_at string
		var updated_at string
		var status string
		var WorkerName string
		var output string

		// Scan the values from the row into variables
		err := rows.Scan(&ID, &created_at, &updated_at, &status, &WorkerName, &output)
		if err != nil {
			fmt.Println(err)
			return tasks, err
		}

		// Data into a Person struct
		var task utils.Task
		task.ID = ID
		task.Created_at = created_at
		task.Updated_at = updated_at
		task.Status = status
		task.WorkerName = WorkerName
		task.Output = output

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

func GetTask(db *sql.DB, ID string) (utils.Task, error) {
	var task utils.Task
	// Retrieve the JSON data from the MySQL table
	var created_at string
	var updated_at string
	var status string
	var WorkerName string
	var output string
	err := db.QueryRow("SELECT ID, created_at, updated_at, status, WorkerName, output FROM task WHERE ID = ?",
		ID).Scan(&ID, &created_at, &updated_at, &status, &WorkerName, &output)
	if err != nil {
		log.Println(err)
		return task, err
	}

	// Data back to a struct
	task.ID = ID
	task.Created_at = created_at
	task.Updated_at = updated_at
	task.Status = status
	task.WorkerName = WorkerName
	task.Output = output

	return task, nil
}

// SetOutputTask save the output of the task in the DB
func SetOutputTask(db *sql.DB, ID string) error {
	fmt.Println("TODO SetOutputTask")
	return nil
}
