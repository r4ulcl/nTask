package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
	structJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJSON := string(structJSON)

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO task (ID, command, name, status, WorkerName, username, priority, callbackURL, callbackToken) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID, commandJSON, task.Name, task.Status, task.WorkerName, task.Username, task.Priority, task.CallbackURL, task.CallbackToken)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask AddTask: ", err)
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
	structJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return err
	}
	commandJSON := string(structJSON)

	// Update all fields in the MySQL table
	_, err = db.Exec("UPDATE task SET command=?, name=?, status=?, WorkerName=?, priority=?, callbackURL=?, callbackToken=? WHERE ID=?",
		commandJSON, task.Name, task.Status, task.WorkerName, task.Priority, task.CallbackURL, task.CallbackToken, task.ID)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask UpdateTask: ", err)
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
		log.Println("DB Delete ID: ", id)
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
func buildFilters(queryParams url.Values) string {
	var filters []string

	addFilter := func(key, format string) {
		value := queryParams.Get(key)
		if value != "" {
			filters = append(filters, fmt.Sprintf(format, key, value))
		}
	}

	addFilter("ID", "ID LIKE '%s'")
	addFilter("command", "command LIKE '%s'")
	addFilter("name", "name LIKE '%s'")
	addFilter("createdAt", "createdAt LIKE '%s'")
	addFilter("updatedAt", "updatedAt LIKE '%s'")
	addFilter("executedAt", "executedAt LIKE '%s'")
	addFilter("status", "status = '%s'")
	addFilter("workerName", "workerName LIKE '%s'")
	addFilter("username", "username LIKE '%s'")
	addFilter("priority", "priority = '%s'")
	addFilter("callbackURL", "callbackURL = '%s'")
	addFilter("callbackToken", "callbackToken = '%s'")

	return strings.Join(filters, " AND ")
}

func buildOrderByAndLimit(page, limit int) string {
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	return fmt.Sprintf(" ORDER BY priority DESC, createdAt ASC LIMIT %d OFFSET %d;", limit, offset)
}

func GetTasks(r *http.Request, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()

	filters := buildFilters(queryParams)
	orderByAndLimit := buildOrderByAndLimit(getPage(queryParams), getLimit(queryParams))

	sql := "SELECT ID, command, name, createdAt, updatedAt, executedAt, status, workerName, username, priority, callbackURL, callbackToken FROM task WHERE 1=1 "

	if filters != "" {
		sql += " AND " + filters
	}

	sql += orderByAndLimit

	if debug {
		log.Println("GetTasks sql", sql)
	}

	return GetTasksSQL(sql, db, verbose, debug)
}

func getPage(queryParams url.Values) int {
	pageStr := queryParams.Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func getLimit(queryParams url.Values) int {
	limitStr := queryParams.Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		return 1000
	}
	return limit
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
			log.Println("DB Error DBTask GetTasksSQL: ", sql, err)
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
				log.Println("DB Error DBTask GetTasksSQL: ", err)
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
			log.Println("DB Error DBTask GetTasksSQL: ", err)
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
			log.Println("DB Error DBTask GetTask: ", err)
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
			log.Println("DB Error DBTask GetTaskExecutedAt: ", err)
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
			log.Println("DB Error DBTask GetTaskWorker: ", err)
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
			log.Println("DB Error DBTask SetTasksWorkerFailed: ", err)
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
			log.Println("DB Error DBTask SetTasksWorkerInvalid: ", err)
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
			log.Println("DB Error DBTask: ", err)
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
			log.Println("DB Error DBTask SetTaskWorkerName: ", err)
		}
		return err
	}
	return nil
}

// SetTasksWorkerEmpty remove the worker name of the task in the database
func SetTasksWorkerEmpty(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the workerName column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET workerName = '' WHERE  workerName = ?", workerName)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask SetTaskWorkerName: ", err)
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
			log.Println("DB Error DBTask SetTaskStatus: ", err)
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
			log.Println("DB Error DBTask SetTaskStatusIfPending: ", err)
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
			log.Println("DB Error DBTask SetTasksStatusIfRunning: ", err)
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
			log.Println("DB Error DBTask SetTaskExecutedAtNow: ", err)
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
			log.Println("DB Error DBTask SetTaskExecutedAt: ", err)
		}
		return err
	}
	return nil
}

// Count

func GetPendingCount(db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query
	query := "SELECT COUNT(*) FROM task where status = 'pending'"

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
	query := "SELECT COUNT(*) FROM task where status = 'running'"

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
	query := "SELECT COUNT(*) FROM task where status = 'done'"

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
	query := "SELECT COUNT(*) FROM task where status = 'failed'"

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
	query := "SELECT COUNT(*) FROM task where status = 'deleted'"

	// Execute the query
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
