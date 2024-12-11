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

	// Convert []files to string and insert
	structJSON, err = json.Marshal(task.Files)
	if err != nil {
		return err
	}
	filesJSON := string(structJSON)

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO task (ID, commands, files, name, status, WorkerName, username, priority, callbackURL, callbackToken) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		task.ID, commandJSON, filesJSON, task.Name, task.Status, task.WorkerName, task.Username, task.Priority, task.CallbackURL, task.CallbackToken)
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

	// Convert []files to string and insert
	structJSON, err = json.Marshal(task.Files)
	if err != nil {
		return err
	}
	filesJSON := string(structJSON)

	// Update all fields in the MySQL table
	_, err = db.Exec("UPDATE task SET commands=?, files=?, name=?, status=?, WorkerName=?, priority=?, callbackURL=?, callbackToken=? WHERE ID=?",
		commandJSON, filesJSON, task.Name, task.Status, task.WorkerName, task.Priority, task.CallbackURL, task.CallbackToken, task.ID)
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

// buildFiltersWithParams constructs SQL filters using query parameters safely.
func buildFiltersWithParams(queryParams url.Values) (string, []interface{}) {
	var filters []string
	var args []interface{}

	addFilter := func(key, condition string) {
		value := queryParams.Get(key)
		if value != "" {
			filters = append(filters, condition)
			args = append(args, value)
		}
	}

	// Apply filters for various fields
	addFilter("ID", "ID LIKE ?")
	addFilter("commands", "commands LIKE ?")
	addFilter("files", "files LIKE ?")
	addFilter("name", "name LIKE ?")
	addFilter("createdAt", "createdAt LIKE ?")
	addFilter("updatedAt", "updatedAt LIKE ?")
	addFilter("executedAt", "executedAt LIKE ?")
	addFilter("status", "status = ?")
	addFilter("workerName", "workerName LIKE ?")
	addFilter("username", "username LIKE ?")
	addFilter("priority", "priority = ?")
	addFilter("callbackURL", "callbackURL = ?")
	addFilter("callbackToken", "callbackToken = ?")

	return strings.Join(filters, " AND "), args
}

func buildOrderByAndLimit(page, limit int) (string, int, int) {
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	orderBy := " ORDER BY priority DESC, createdAt ASC"
	return orderBy, limit, offset
}

// GetTasks retrieves tasks from the database using URL parameters as filters.
func GetTasks(r *http.Request, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()

	// Parse filters and build SQL conditions
	filters, args := buildFiltersWithParams(queryParams)

	// Build ORDER BY and LIMIT clauses
	orderBy, limit, offset := buildOrderByAndLimit(getPage(queryParams), getLimit(queryParams))

	// Base SQL query
	sql := "SELECT ID, commands, files, name, createdAt, updatedAt, executedAt, status, workerName, username, priority, callbackURL, callbackToken FROM task WHERE 1=1"

	// Append filters
	if filters != "" {
		sql += " AND " + filters
	}

	// Add ORDER BY clause
	sql += orderBy

	// Add LIMIT and OFFSET as placeholders
	sql += " LIMIT ? OFFSET ?"

	// Append LIMIT and OFFSET to arguments
	args = append(args, limit, offset)

	if debug {
		log.Println("GetTasks SQL:", sql)
		log.Println("Query Arguments:", args)
	}

	// Fetch tasks using the constructed SQL query
	return GetTasksSQL(sql, args, db, verbose, debug)
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

// GetTasksPending Get Tasks  with status = Pending
func GetTasksPending(limit int, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	sql := "SELECT ID, commands, files, name, createdAt, updatedAt, executedAt, status, WorkerName, username, priority, callbackURL, callbackToken FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC LIMIT ?"
	return GetTasksSQL(sql, []interface{}{limit}, db, verbose, debug)
}

// GetTasksSQL executes a parameterized SQL query to fetch tasks.
func GetTasksSQL(sqlQuery string, args []interface{}, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	var tasks []globalstructs.Task

	// Execute the parameterized query
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask GetTasksSQL:", sqlQuery, err)
		}
		return tasks, err
	}
	defer rows.Close()

	// Process rows and map to task objects
	for rows.Next() {
		var task globalstructs.Task
		var commandsAux, filesAux string

		// Scan row values into variables
		err := rows.Scan(
			&task.ID, &commandsAux, &filesAux, &task.Name,
			&task.CreatedAt, &task.UpdatedAt, &task.ExecutedAt, &task.Status,
			&task.WorkerName, &task.Username, &task.Priority, &task.CallbackURL,
			&task.CallbackToken,
		)
		if err != nil {
			if debug {
				log.Println("DB Error DBTask GetTasksSQL: Row Scan", err)
			}
			return tasks, err
		}

		// Convert JSON strings to slices of structs
		if err := json.Unmarshal([]byte(commandsAux), &task.Commands); err != nil {
			return tasks, fmt.Errorf("error parsing commands: %w", err)
		}
		if err := json.Unmarshal([]byte(filesAux), &task.Files); err != nil {
			return tasks, fmt.Errorf("error parsing files: %w", err)
		}

		// Append task to the results slice
		tasks = append(tasks, task)
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		if debug {
			log.Println("DB Error DBTask GetTasksSQL: Rows Iteration", err)
		}
		return tasks, err
	}

	return tasks, nil
}

// GetTask gets task filtered by id
func GetTask(db *sql.DB, id string, verbose, debug bool) (globalstructs.Task, error) {
	var task globalstructs.Task
	// Retrieve the JSON data from the MySQL table
	var commandsAux string
	var filesAux string
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

	err := db.QueryRow("SELECT ID, createdAt, updatedAt, executedAt, commands, files, name, status, WorkerName, username, priority, callbackURL, callbackToken FROM task WHERE ID = ?",
		id).Scan(&id, &createdAt, &updatedAt, &executedAt, &commandsAux, &filesAux, &name, &status, &workerName, &username, &priority, &callbackURL, &callbackToken)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask GetTask: ", err)
		}
		return task, err
	}

	// Data back to a struct
	task.ID = id
	// String to []struct
	var commands []globalstructs.Command
	err = json.NewDecoder(strings.NewReader(commandsAux)).Decode(&commands)
	if err != nil {
		return task, err
	}
	task.Commands = commands
	// String to []struct
	var files []globalstructs.File
	err = json.NewDecoder(strings.NewReader(filesAux)).Decode(&files)
	if err != nil {
		return task, err
	}
	task.Files = files
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

// GetTaskExecutedAt Get Task ExecutedAt by id
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

/*
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
}*/

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

// setTasksWorkerEmpty remove the worker name of the task in the database
func setTasksWorkerEmpty(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	// Add to the WaitGroup when the goroutine starts and done when exits
	defer wg.Done()
	wg.Add(1)
	// Update the workerName column of the task table for the given ID
	_, err := db.Exec("UPDATE task SET workerName = '' WHERE  workerName = ?", workerName)
	if err != nil {
		if debug || verbose {
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

/*
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
*/

// SetTasksStatusIfRunning saves the status of the task in the database if current is running
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

// SetTaskExecutedAtNow saves current time as executedAt
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

/*
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
*/
// Count

// GetPendingCount Get count of pending tasks
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

// GetRunningCount count of running tasks
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

// GetDoneCount count of done tasks
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

// GetFailedCount count of failed tasks
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

// GetDeletedCount count of deleted tasks
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
