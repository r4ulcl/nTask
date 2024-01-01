package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// AddTask adds a task to the database
func AddTask(db *sql.DB, task globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Convert []command to string and insert
	structJson, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJson := string(structJson)

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO task (ID, command, name, status, WorkerName, username, priority, callbackURL, callbackToken) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID, commandJson, task.Name, task.Status, task.WorkerName, task.Username, task.Priority, task.CallbackURL, task.CallbackToken)
	if err != nil {
		if debug {
			log.Println("Error DBTask AddTask: ", err)
		}
		return err
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)

	// Convert []command to string and insert
	structJson, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJson := string(structJson)

	// Update all fields in the MySQL table
	_, err = db.Exec("UPDATE task SET command=?, name=?, status=?, WorkerName=?, priority=?, callbackURL=?, callbackToken=? WHERE ID=?",
		commandJson, task.Name, task.Status, task.WorkerName, task.Priority, task.CallbackURL, task.CallbackToken, task.ID)
	if err != nil {
		if debug {
			log.Println("Error DBTask UpdateTask: ", err)
		}
		return err
	}

	return nil
}

// RmTask deletes a task from the database.
func RmTask(db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM task WHERE ID LIKE ?"
	if debug {
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
func GetTasks(w http.ResponseWriter, r *http.Request, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()

	sql := "SELECT ID, command, name, createdAt, updatedAt, executedAt, status, workerName, username, priority, callbackURL, callbackToken FROM task WHERE 1=1 "

	// Add filters for each parameter if provided
	if ID := queryParams.Get("ID"); ID != "" {
		sql += fmt.Sprintf(" AND ID LIKE  '%s'", ID)
	}

	if command := queryParams.Get("command"); command != "" {
		sql += fmt.Sprintf(" AND command LIKE '%s'", command)
	}

	if name := queryParams.Get("name"); name != "" {
		sql += fmt.Sprintf(" AND name LIKE '%s'", name)
	}

	if createdAt := queryParams.Get("createdAt"); createdAt != "" {
		sql += fmt.Sprintf(" AND createdAt LIKE '%s'", createdAt)
	}

	if updatedAt := queryParams.Get("executedAt"); updatedAt != "" {
		sql += fmt.Sprintf(" AND executedAt LIKE '%s'", updatedAt)
	}

	if updatedAt := queryParams.Get("updatedAt"); updatedAt != "" {
		sql += fmt.Sprintf(" AND updatedAt LIKE '%s'", updatedAt)
	}

	if status := queryParams.Get("status"); status != "" {
		sql += fmt.Sprintf(" AND status = '%s'", status)
	}

	if workerName := queryParams.Get("workerName"); workerName != "" {
		sql += fmt.Sprintf(" AND workerName LIKE '%s'", workerName)
	}

	if username := queryParams.Get("username"); username != "" {
		sql += fmt.Sprintf(" AND username LIKE '%s'", username)
	}

	if priority := queryParams.Get("priority"); priority != "" {
		sql += fmt.Sprintf(" AND priority = '%s'", priority)
	}

	if callbackURL := queryParams.Get("callbackURL"); callbackURL != "" {
		sql += fmt.Sprintf(" AND callbackURL = '%s'", callbackURL)
	}

	if callbackToken := queryParams.Get("callbackToken"); callbackToken != "" {
		sql += fmt.Sprintf(" AND callbackToken = '%s'", callbackToken)
	}

	if limit := queryParams.Get("limit"); limit != "" {
		sql += fmt.Sprintf(" limit '%s'", limit)
	}

	sql += " ORDER BY priority DESC, createdAt ASC;"
	return GetTasksSQL(sql, db, verbose, debug)
}

// GetTasksPending gets only tasks with status pending
func GetTasksPending(limit int, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	sql := "SELECT ID, command, name, createdAt, updatedAt, executedAt, status, WorkerName, username, " +
		"priority, callbackURL, callbackToken FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC limit %d"
	formattedSQL := fmt.Sprintf(sql, limit)
	return GetTasksSQL(formattedSQL, db, verbose, debug)
}

// GetTasksSQL gets tasks by passing the SQL query in sql param
func GetTasksSQL(sql string, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	var tasks []globalstructs.Task

	// Query all tasks from the task table
	rows, err := db.Query(sql)
	if err != nil {
		if debug {
			log.Println("Error DBTask GetTasksSQL: ", sql, err)
		}
		return tasks, err
	}
	defer rows.Close()

	// Iterate over the rows
	for rows.Next() {
		// Declare variables to store JSON data
		var ID string
		var commandAux string
		var name string
		var createdAt string
		var updatedAt string
		var executedAt string
		var status string
		var workerName string
		var username string
		var priority int
		var callbackURL string
		var callbackToken string

		// Scan the values from the row into variables
		err := rows.Scan(&ID, &commandAux, &name, &createdAt, &updatedAt, &executedAt, &status, &workerName, &username, &priority, &callbackURL, &callbackToken)
		if err != nil {
			if debug {
				log.Println("Error DBTask GetTasksSQL: ", err)
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
		task.Name = name
		task.CreatedAt = createdAt
		task.UpdatedAt = updatedAt
		task.ExecutedAt = executedAt
		task.Status = status
		task.WorkerName = workerName
		task.Username = username
		task.Priority = priority
		task.CallbackURL = callbackURL
		task.CallbackToken = callbackToken

		// Append the task to the slice
		tasks = append(tasks, task)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		if debug {
			log.Println("Error DBTask GetTasksSQL: ", err)
		}
		return tasks, err
	}

	return tasks, nil
}

// GetTask gets task filtered by id
func GetTask(db *sql.DB, id string, verbose, debug bool) (globalstructs.Task, error) {
	var task globalstructs.Task
	// Retrieve the JSON data from the MySQL table
	var commandAux string
	var name string
	var createdAt string
	var updatedAt string
	var executedAt string
	var status string
	var workerName string
	var username string
	var priority int
	var callbackURL string
	var callbackToken string

	err := db.QueryRow("SELECT ID, createdAt, updatedAt, executedAt, command, name, status, WorkerName, username, priority, callbackURL, callbackToken FROM task WHERE ID = ?",
		id).Scan(&id, &createdAt, &updatedAt, &executedAt, &commandAux, &name, &status, &workerName, &username, &priority, &callbackURL, &callbackToken)
	if err != nil {
		if debug {
			log.Println("Error DBTask GetTask: ", err)
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
	task.Name = name
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt
	task.ExecutedAt = executedAt
	task.Status = status
	task.WorkerName = workerName
	task.Username = username
	task.Priority = priority
	task.CallbackURL = callbackURL
	task.CallbackToken = callbackToken

	return task, nil
}

// GetTaskExecutedAt
func GetTaskExecutedAt(db *sql.DB, id string, verbose, debug bool) (string, error) {
	// Retrieve the workerName from the task table
	var executedAt string
	err := db.QueryRow("SELECT executedAt FROM task WHERE ID = ?",
		id).Scan(&executedAt)
	if err != nil {
		if debug {
			log.Println("Error DBTask GetTaskExecutedAt: ", err)
		}
		return executedAt, err
	}

	return executedAt, nil
}

// GetTaskWorker gets task workerName from an ID
// This is the worker executing the task
func GetTaskWorker(db *sql.DB, id string, verbose, debug bool) (string, error) {
	// Retrieve the workerName from the task table
	var workerName string
	err := db.QueryRow("SELECT WorkerName FROM task WHERE ID = ?",
		id).Scan(&workerName)
	if err != nil {
		if debug {
			log.Println("Error DBTask GetTaskWorker: ", err)
		}
		return workerName, err
	}

	return workerName, nil
}

// SetTasksWorkerFailed set to failed all task running worker workerName
func SetTasksWorkerFailed(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE task SET status = 'failed' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTasksWorkerFailed: ", err)
		}
		return err
	}
	return nil
}

// SetTasksWorkerInvalid set to invalid all task running worker workerName
func SetTasksWorkerInvalid(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE task SET status = 'invalid' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTasksWorkerInvalid: ", err)
		}
		return err
	}
	return nil
}

// SetTasksWorkerPending set all task of worker to pending because failed
func SetTasksWorkerPending(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE task SET status = 'pending' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if debug {
			log.Println("Error DBTask: ", err)
		}
		return err
	}
	return nil
}

// SetTaskWorkerName saves the worker name of the task in the database
func SetTaskWorkerName(db *sql.DB, id, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the workerName column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET workerName = ? WHERE ID = ?", workerName, id)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTaskWorkerName: ", err)
		}
		return err
	}
	return nil
}

// SetTaskStatus saves the status of the task in the database
func SetTaskStatus(db *sql.DB, id, status string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET status = ? WHERE ID = ?", status, id)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTaskStatus: ", err)
		}
		return err
	}
	return nil
}

// SetTaskStatusIfPending saves the status of the task in the database if current is pending
func SetTaskStatusIfPending(db *sql.DB, id, status string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET status = ? WHERE ID = ? and status = 'pending'", status, id)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTaskStatusIfPending: ", err)
		}
		return err
	}
	return nil
}

// SetTaskStatusIfPending saves the status of the task in the database if current is pending
func SetTasksStatusIfRunning(db *sql.DB, status string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET status = ? WHERE status = 'running'", status)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTasksStatusIfRunning: ", err)
		}
		return err
	}
	return nil
}

// SetTaskExecutedAt saves current time as executedAt
func SetTaskExecutedAtNow(db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET executedAt = now() WHERE ID = ?", id)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTaskExecutedAtNow: ", err)
		}
		return err
	}
	return nil
}

// SetTaskExecutedAt saves current time as executedAt
func SetTaskExecutedAt(executedAt string, db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the status column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET executedAt = ? WHERE ID = ?", executedAt, id)
	if err != nil {
		if debug {
			log.Println("Error DBTask SetTaskExecutedAt: ", err)
		}
		return err
	}
	return nil
}

// Count

func GetPendingCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := fmt.Sprintf("SELECT COUNT(*) FROM task where status = 'pending'")

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetRunningCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := fmt.Sprintf("SELECT COUNT(*) FROM task where status = 'running'")

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetDoneCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := fmt.Sprintf("SELECT COUNT(*) FROM task where status = 'done'")

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetFailedCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := fmt.Sprintf("SELECT COUNT(*) FROM task where status = 'failed'")

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetDeletedCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := fmt.Sprintf("SELECT COUNT(*) FROM task where status = 'deleted'")

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
