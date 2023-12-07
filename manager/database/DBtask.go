package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

// AddTask adds a task to the database
func AddTask(db *sql.DB, task globalstructs.Task, verbose bool) error {
	// Convert []command to string and insert
	structJson, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJson := string(structJson)

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO task (ID, command, status, WorkerName, priority) VALUES (?, ?, ?, ?, ?)",
		task.ID, commandJson, task.Status, task.WorkerName, task.Priority)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalstructs.Task, verbose bool) error {
	// Convert []command to string and insert
	structJson, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJson := string(structJson)

	// Update all fields in the MySQL table
	_, err = db.Exec("UPDATE task SET command=?, status=?, WorkerName=?, priority=? WHERE ID=?",
		commandJson, task.Status, task.WorkerName, task.Priority, task.ID)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// RmTask deletes a task from the database.
func RmTask(db *sql.DB, id string, verbose bool) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM task WHERE ID = ?"
	if verbose {
		log.Println("Delete ID: ", id)
	}
	result, err := db.Exec(sqlStatement, id)
	if err != nil {
		return err
	}

	a, _ := result.RowsAffected()

	if a < 1 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// GetTasks gets tasks with URL params as filter.
func GetTasks(w http.ResponseWriter, r *http.Request, db *sql.DB, verbose bool) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()

	sql := "SELECT ID, command, createdAt, updatedAt, status, workerName, priority FROM task WHERE 1=1 "

	// Add filters for each parameter if provided
	if ID := queryParams.Get("ID"); ID != "" {
		sql += fmt.Sprintf(" AND ID = '%s'", ID)
	}

	if command := queryParams.Get("command"); command != "" {
		sql += fmt.Sprintf(" AND command = '%s'", command)
	}

	if createdAt := queryParams.Get("createdAt"); createdAt != "" {
		sql += fmt.Sprintf(" AND createdAt = '%s'", createdAt)
	}

	if updatedAt := queryParams.Get("updatedAt"); updatedAt != "" {
		sql += fmt.Sprintf(" AND updatedAt = '%s'", updatedAt)
	}

	if status := queryParams.Get("status"); status != "" {
		sql += fmt.Sprintf(" AND status = '%s'", status)
	}

	if workerName := queryParams.Get("workerName"); workerName != "" {
		sql += fmt.Sprintf(" AND workerName = '%s'", workerName)
	}

	if priority := queryParams.Get("priority"); priority != "" {
		sql += fmt.Sprintf(" AND priority = '%s'", priority)
	}
	sql += " ORDER BY priority DESC, createdAt ASC;"

	return GetTasksSQL(sql, db, verbose)
}

// GetTasksPending gets only tasks with status pending
func GetTasksPending(db *sql.DB, verbose bool) ([]globalstructs.Task, error) {
	sql := "SELECT ID, command, createdAt, updatedAt, status, WorkerName, priority FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC"
	return GetTasksSQL(sql, db, verbose)
}

// GetTasksSQL gets tasks by passing the SQL query in sql param
func GetTasksSQL(sql string, db *sql.DB, verbose bool) ([]globalstructs.Task, error) {
	var tasks []globalstructs.Task

	// Query all tasks from the task table
	rows, err := db.Query(sql)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return tasks, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var ID string
		var commandAux string
		var createdAt string
		var updatedAt string
		var status string
		var WorkerName string
		var priority bool

		// Scan the values from the row into variables
		err := rows.Scan(&ID, &commandAux, &createdAt, &updatedAt, &status, &WorkerName, &priority)
		if err != nil {
			if verbose {
				log.Println(err)
			}
			return tasks, err
		}

		// Data into a Task struct
		var task globalstructs.Task
		task.ID = ID

		// String to []struct
		var command []globalstructs.Command
		err = json.NewDecoder(strings.NewReader(commandAux)).Decode(&command)
		if err != nil {
			return tasks, err
		}
		task.Commands = command
		task.CreatedAt = createdAt
		task.UpdatedAt = updatedAt
		task.Status = status
		task.WorkerName = WorkerName
		task.Priority = priority

		// Append the task to the slice
		tasks = append(tasks, task)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		if verbose {
			log.Println(err)
		}
		return tasks, err
	}

	return tasks, nil
}

// GetTask gets task filtered by id
func GetTask(db *sql.DB, id string, verbose bool) (globalstructs.Task, error) {
	var task globalstructs.Task
	// Retrieve the JSON data from the MySQL table
	var commandAux string
	var createdAt string
	var updatedAt string
	var status string
	var WorkerName string
	var priority bool

	err := db.QueryRow("SELECT ID, createdAt, updatedAt, command, status, WorkerName, priority FROM task WHERE ID = ?",
		id).Scan(&id, &createdAt, &updatedAt, &commandAux, &status, &WorkerName, &priority)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return task, err
	}

	// Data back to a struct
	task.ID = id
	// String to []struct
	var command []globalstructs.Command
	err = json.NewDecoder(strings.NewReader(commandAux)).Decode(&command)
	if err != nil {
		return task, err
	}
	task.Commands = command
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt
	task.Status = status
	task.WorkerName = WorkerName
	task.Priority = priority

	return task, nil
}

// GetTaskWorker gets task workerName from an ID
// This is the worker executing the task
func GetTaskWorker(db *sql.DB, id string, verbose bool) (string, error) {
	// Retrieve the workerName from the task table
	var workerName string
	err := db.QueryRow("SELECT WorkerName FROM task WHERE ID = ?",
		id).Scan(&workerName)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return workerName, err
	}

	return workerName, nil
}

//SetTasksWorkerFailed set to failed all task running worker workerName
func SetTasksWorkerFailed(db *sql.DB, workerName string, verbose bool) error {
	_, err := db.Exec("UPDATE task SET status = 'failed' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// SetTaskWorkerName saves the worker name of the task in the database
func SetTaskWorkerName(db *sql.DB, id, workerName string, verbose bool) error {
	// Update the workerName column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET workerName = ? WHERE ID = ?", workerName, id)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}

// SetTaskStatus saves the status of the task in the database
func SetTaskStatus(db *sql.DB, id, status string, verbose bool) error {
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET status = ? WHERE ID = ?", status, id)
	if err != nil {
		if verbose {
			log.Println(err)
		}
		return err
	}
	return nil
}
