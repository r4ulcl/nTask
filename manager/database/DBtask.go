package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	globalstructs "github.com/r4ulcl/NetTask/globalstructs"
)

func AddTask(db *sql.DB, task globalstructs.Task) error {
	// Insert the JSON data into the MySQL table
	argsString := strings.Join(task.Args, ",")
	_, err := db.Exec("INSERT INTO task (ID, module, args, status, WorkerName, output) VALUES (?, ?, ?, ?, ?, ?)",
		task.ID, task.Module, argsString, task.Status, task.WorkerName, task.Output)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalstructs.Task) error {
	// Update all fields in the MySQL table
	argsString := strings.Join(task.Args, ",")
	_, err := db.Exec("UPDATE task SET module=?, args=?, status=?, WorkerName=?, output=? WHERE ID=?",
		task.Module, argsString, task.Status, task.WorkerName, task.Output, task.ID)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func RmTask(db *sql.DB, id string) error {
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM task WHERE ID = ?"
	log.Println("ID: ", id)
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

func GetTasks(w http.ResponseWriter, r *http.Request, db *sql.DB) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()

	sql := "SELECT ID, module, args, createdAt, updatedAt, status, workerName, output, priority FROM task WHERE 1=1 "

	// Add filters for each parameter if provided
	if ID := queryParams.Get("ID"); ID != "" {
		sql += fmt.Sprintf(" AND ID = '%s'", ID)
	}

	if module := queryParams.Get("module"); module != "" {
		sql += fmt.Sprintf(" AND module = '%s'", module)
	}

	if args := queryParams.Get("args"); args != "" {
		sql += fmt.Sprintf(" AND args = '%s'", args)
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

	if output := queryParams.Get("output"); output != "" {
		sql += fmt.Sprintf(" AND output = '%s'", output)
	}

	if priority := queryParams.Get("priority"); priority != "" {
		sql += fmt.Sprintf(" AND priority = '%s'", priority)
	}
	sql += " ORDER BY priority DESC, createdAt ASC;"

	// log.Println(sql)
	return GetTasksSQL(sql, db)
}

func GetTasks2(db *sql.DB) ([]globalstructs.Task, error) {
	sql := "SELECT ID, module, args, createdAt, updatedAt, status, WorkerName, output, priority FROM task ORDER BY priority DESC, createdAt ASC"
	return GetTasksSQL(sql, db)
}

func GetTasksPending(db *sql.DB) ([]globalstructs.Task, error) {
	sql := "SELECT ID, module, args, createdAt, updatedAt, status, WorkerName, output, priority FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC"
	return GetTasksSQL(sql, db)
}

func GetTasksSQL(sql string, db *sql.DB) ([]globalstructs.Task, error) {
	var tasks []globalstructs.Task

	// Query all tasks from the task table
	rows, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		return tasks, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var ID string
		var module string
		var args string
		var createdAt string
		var updatedAt string
		var status string
		var WorkerName string
		var output string
		var priority bool

		// Scan the values from the row into variables
		err := rows.Scan(&ID, &module, &args, &createdAt, &updatedAt, &status, &WorkerName, &output, &priority)
		if err != nil {
			log.Println(err)
			return tasks, err
		}

		// Data into a Person struct
		var task globalstructs.Task
		task.ID = ID
		task.Module = module
		task.Args = strings.Split(args, ",")
		task.CreatedAt = createdAt
		task.UpdatedAt = updatedAt
		task.Status = status
		task.WorkerName = WorkerName
		task.Output = output
		task.Priority = priority

		// Append the person to the slice
		tasks = append(tasks, task)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		log.Println(err)
		return tasks, err
	}

	return tasks, nil
}

func GetTask(db *sql.DB, id string) (globalstructs.Task, error) {
	var task globalstructs.Task
	// Retrieve the JSON data from the MySQL table
	var module string
	var args string
	var createdAt string
	var updatedAt string
	var status string
	var WorkerName string
	var output string
	err := db.QueryRow("SELECT ID, createdAt, updatedAt, module, args, status, WorkerName, output FROM task WHERE ID = ?",
		id).Scan(&id, &createdAt, &updatedAt, &module, &args, &status, &WorkerName, &output)
	if err != nil {
		log.Println(err)
		return task, err
	}

	// Data back to a struct
	task.ID = id
	task.Module = module
	task.Args = strings.Split(args, ",")
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt
	task.Status = status
	task.WorkerName = WorkerName
	task.Output = output

	return task, nil
}

func GetTaskWorker(db *sql.DB, id string) (string, error) {
	// Retrieve the JSON data from the MySQL table
	var workerName string
	err := db.QueryRow("SELECT WorkerName FROM task WHERE ID = ?",
		id).Scan(&workerName)
	if err != nil {
		log.Println(err)
		return workerName, err
	}

	return workerName, nil
}

// SetTaskOutput save the output of the task in the DB
func SetTaskOutput(db *sql.DB, id, output string) error {
	_, err := db.Exec("UPDATE task SET output = ? WHERE ID = ?",
		output, id)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// SetTaskWorkerName save the output of the task in the DB
func SetTaskWorkerName(db *sql.DB, id, workerName string) error {
	_, err := db.Exec("UPDATE task SET workerName = ? WHERE ID = ?",
		workerName, id)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// SetTaskStatus save the output of the task in the DB
func SetTaskStatus(db *sql.DB, id, status string) error {
	_, err := db.Exec("UPDATE task SET status = ? WHERE ID = ?",
		status, id)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
