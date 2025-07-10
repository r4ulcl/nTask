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
	"time"

	"github.com/go-sql-driver/mysql"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// execWithRetry wraps db.Exec to retry on MySQL deadlock (Error 1213).
func execWithRetry(db *sql.DB, wg *sync.WaitGroup, query string, args ...interface{}) (sql.Result, error) {
	wg.Add(1)
	defer wg.Done()
	const maxRetries = 3
	var err error
	for i := 0; i < maxRetries; i++ {
		var result sql.Result
		result, err = db.Exec(query, args...)
		if err == nil {
			return result, nil
		}
		// Retry on deadlock
		if merr, ok := err.(*mysql.MySQLError); ok && merr.Number == 1213 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("deadlock after %d retries for query %q: %q", maxRetries, query, err)
}

// serializeToJSON marshals a slice into a JSON string.
func serializeToJSON(data interface{}) (string, error) {
	structJSON, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(structJSON), nil
}

// prepareTaskQuery prepare task insertion or update in the database.
func prepareTaskQuery(task globalstructs.Task, verbose, debug bool) (string, string, error) {
	commandJSON, err := serializeToJSON(task.Commands)
	if err != nil {
		return "", "", err
	}

	filesJSON, err := serializeToJSON(task.Files)
	if err != nil {
		return "", "", err
	}

	if debug || verbose {
		log.Println("prepareTaskQuery: filesJSON - ", filesJSON, " - commandJSON", commandJSON)
	}

	return commandJSON, filesJSON, nil
}

// AddTask adds a task to the database.
func AddTask(db *sql.DB, task globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup) error {
	query := "INSERT INTO task (ID, notes, commands, files, name, status, duration, WorkerName, username, priority, timeout, callbackURL, callbackToken) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	commandJSON, filesJSON, err := prepareTaskQuery(task, verbose, debug)
	if err != nil {
		return err
	}
	_, err = execWithRetry(db, wg, query, task.ID, task.Notes, commandJSON, filesJSON, task.Name, task.Status, task.Duration, task.WorkerName, task.Username, task.Priority, task.Timeout, task.CallbackURL, task.CallbackToken)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask Query Execution: ", err)
		}
		return err
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalstructs.Task, verbose, debug bool, wg *sync.WaitGroup) error {

	if debug {
		log.Println("updating Task", task)
	}

	query := "UPDATE task SET notes=?, commands=?, files=?, name=?, status=?, duration=?, WorkerName=?, priority=?, timeout=?, callbackURL=?, callbackToken=? WHERE ID=?"
	commandJSON, filesJSON, err := prepareTaskQuery(task, verbose, debug)
	if err != nil {
		return err
	}

	_, err = execWithRetry(db, wg, query,
		task.Notes, commandJSON, filesJSON, task.Name,
		task.Status, task.Duration, task.WorkerName,
		task.Priority, task.Timeout, task.CallbackURL,
		task.CallbackToken, task.ID,
	)

	if err != nil {
		if debug {
			log.Println("DB Error DBTask Query Execution: ", err)
		}
		return err
	}
	return nil
}

// RmTask deletes a task from the database.
func RmTask(db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {

	// Worker exists, proceed with deletion
	sqlStatement := "DELETE FROM task WHERE ID LIKE ?"
	if debug {
		log.Println("DB Delete ID: ", id)
	}

	result, err := execWithRetry(db, wg, sqlStatement, id)
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
	addFilter("notes", "notes LIKE ?")
	addFilter("commands", "commands LIKE ?")
	addFilter("files", "files LIKE ?")
	addFilter("name", "name LIKE ?")
	addFilter("createdAt", "createdAt LIKE ?")
	addFilter("updatedAt", "updatedAt LIKE ?")
	addFilter("executedAt", "executedAt LIKE ?")
	addFilter("status", "status = ?")
	addFilter("duration", "duration = ?")
	addFilter("workerName", "workerName LIKE ?")
	addFilter("username", "username LIKE ?")
	addFilter("priority", "priority = ?")
	addFilter("timeout", "timeout = ?")
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
	sql := "SELECT ID, notes, commands, files, name, createdAt, updatedAt, executedAt, status, duration, workerName, username, priority, timeout, callbackURL, callbackToken FROM task WHERE 1=1"

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
	return getTasksSQL(sql, args, db, verbose, debug)
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
	sql := "SELECT ID, notes, commands, files, name, createdAt, updatedAt, executedAt, status, duration, WorkerName, username, priority, timeout, callbackURL, callbackToken FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC LIMIT ?"
	return getTasksSQL(sql, []interface{}{limit}, db, verbose, debug)
}

// getTasksSQL executes a parameterized SQL query to fetch tasks.
func getTasksSQL(sqlQuery string, args []interface{}, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	var tasks []globalstructs.Task

	// Execute the parameterized query
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask getTasksSQL:", sqlQuery, err)
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
			&task.ID, &task.Notes, &commandsAux, &filesAux, &task.Name,
			&task.CreatedAt, &task.UpdatedAt, &task.ExecutedAt, &task.Status,
			&task.Duration, &task.WorkerName, &task.Username, &task.Priority,
			&task.Timeout, &task.CallbackURL, &task.CallbackToken,
		)
		if err != nil {
			if debug {
				log.Println("DB Error DBTask getTasksSQL: Row Scan", err)
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
			log.Println("DB Error DBTask getTasksSQL: Rows Iteration", err)
		}
		return tasks, err
	}

	return tasks, nil
}

// GetTask gets task filtered by id
func GetTask(db *sql.DB, id string, verbose, debug bool) (globalstructs.Task, error) {
	var task globalstructs.Task
	// Retrieve the JSON data from the MySQL table
	var notes string
	var commandsAux string
	var filesAux string
	var name string
	var createdAt string
	var updatedAt string
	var executedAt string
	var status string
	var duration float64
	var workerName string
	var username string
	var priority int
	var timeout int
	var callbackURL string
	var callbackToken string

	err := db.QueryRow("SELECT ID, notes, createdAt, updatedAt, executedAt, commands, files, name, status, duration, workerName, username, priority, timeout, callbackURL, callbackToken FROM task WHERE ID = ?",
		id).Scan(&id, &notes, &createdAt, &updatedAt, &executedAt, &commandsAux, &filesAux, &name, &status, &duration, &workerName, &username, &priority, &timeout, &callbackURL, &callbackToken)
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
	task.Notes = notes
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt
	task.ExecutedAt = executedAt
	task.Status = status
	task.Duration = duration
	task.WorkerName = workerName
	task.Username = username
	task.Priority = priority
	task.Timeout = timeout
	task.CallbackURL = callbackURL
	task.CallbackToken = callbackToken

	return task, nil
}

/*
// getTaskExecutedAt Get Task ExecutedAt by id
func getTaskExecutedAt(db *sql.DB, id string, verbose, debug bool) (string, error) {
	// Retrieve the workerName from the task table
	var executedAt string
	err := db.QueryRow("SELECT executedAt FROM task WHERE ID = ?",
		id).Scan(&executedAt)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask getTaskExecutedAt: ", err)
		}
		return executedAt, err
	}

	return executedAt, nil
}


// getTaskWorker gets task workerName from an ID
// This is the worker executing the task
func getTaskWorker(db *sql.DB, id string, verbose, debug bool) (string, error) {
	// Retrieve the workerName from the task table
	var workerName string
	err := db.QueryRow("SELECT WorkerName FROM task WHERE ID = ?",
		id).Scan(&workerName)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask getTaskWorker: ", err)
		}
		return workerName, err
	}

	return workerName, nil
}

// setTasksWorkerFailed set to failed all task running worker workerName
func setTasksWorkerFailed(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE task SET status = 'failed' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask setTasksWorkerFailed: ", err)
		}
		return err
	}
	return nil
}

// setTasksWorkerInvalid set to invalid all task running worker workerName
func setTasksWorkerInvalid(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	_, err := db.Exec("UPDATE task SET status = 'invalid' WHERE workerName = ? AND status = 'running' ", workerName)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask setTasksWorkerInvalid: ", err)
		}
		return err
	}
	return nil
}*/

// Generic helper function to execute a database update
func executeDBUpdate(db *sql.DB, query string, args []interface{}, verbose, debug bool, wg *sync.WaitGroup, taskName string) error {
	_, err := execWithRetry(db, wg, query, args...)
	if err != nil {
		if debug || verbose {
			log.Printf("DB Error %s: %v", taskName, err)
		}
		return err
	}
	return nil
}

// SetTasksWorkerPending Function to set tasks worker status to 'pending'
func SetTasksWorkerPending(db *sql.DB, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {
	query := "UPDATE task SET status = 'pending' WHERE workerName = ? AND status = 'running'"
	args := []interface{}{workerName}
	return executeDBUpdate(db, query, args, verbose, debug, wg, "DBTask: SetTasksWorkerPending")
}

// SetTaskExecutedAtNow Function to set task's executedAt timestamp to now()
func SetTaskExecutedAtNow(db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {
	query := "UPDATE task SET executedAt = now() WHERE ID = ?"
	args := []interface{}{id}
	return executeDBUpdate(db, query, args, verbose, debug, wg, "DBTask SetTaskExecutedAtNow")
}

// SetTaskWorkerName saves the worker name of the task in the database
func SetTaskWorkerName(db *sql.DB, id, workerName string, verbose, debug bool, wg *sync.WaitGroup) error {

	// Update the workerName column of the task table for the given ID
	query := "UPDATE task SET workerName = ? WHERE ID = ?"
	_, err := execWithRetry(db, wg, query, workerName, id)
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

	// Update the workerName column of the task table for the given ID
	query := "UPDATE task SET workerName = '' WHERE  workerName = ?"
	_, err := execWithRetry(db, wg, query, workerName)
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

	if debug {
		log.Println("SetTaskStatus", status, id)
	}

	// Update the status column of the task table for the given ID
	query := "UPDATE task SET status = ? WHERE ID = ?"
	_, err := execWithRetry(db, wg, query, status, id)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask SetTaskStatus: ", err)
		}
		return err
	}
	return nil
}

// SetTasksStatusIfStatus saves the status of the task in the database if current status is currentStatus
func SetTasksStatusIfStatus(currentStatus string, db *sql.DB, newStatus string, verbose, debug bool, wg *sync.WaitGroup) error {

	// Update the status column of the task table for the given ID
	query := "UPDATE task SET status = ? WHERE status = ?"
	_, err := execWithRetry(db, wg, query, newStatus, currentStatus)
	if err != nil {
		if debug || verbose {
			log.Println("DB Error DBTask SetTasksStatusIfRunning: ", err)
		}
		return err
	}
	return nil
}

// SetTaskExecutedAt saves current time as executedAt
func SetTaskExecutedAt(executedAt string, db *sql.DB, id string, verbose, debug bool, wg *sync.WaitGroup) error {

	// Update the status column of the task table for the given ID
	query := "UPDATE task SET executedAt = ? WHERE status = ?"
	_, err := execWithRetry(db, wg, query, executedAt, id)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask SetTaskExecutedAt: ", err)
		}
		return err
	}
	return nil
}

// Count

// GetCountByStatus get count of tasks with status X
func GetCountByStatus(status string, db *sql.DB, verbose, debug bool) (int, error) {
	// Prepare the SQL query with a placeholder for the status
	query := "SELECT COUNT(*) FROM task WHERE status = ?"

	if debug {
		log.Println("Executing GetCountByStatus")
	}

	// Execute the query with the provided status
	var count int
	err := db.QueryRow(query, status).Scan(&count)
	if err != nil {
		return 0, err
	}

	if verbose || debug {
		log.Printf("GetCountByStatus for status '%s': %d\n", status, count)
	}

	return count, nil
}

// DeleteMaxEntriesHistory Delete Database Entries if entries number > maxEntries
func DeleteMaxEntriesHistory(db *sql.DB, maxEntries int, tableName string, verbose, debug bool, wg *sync.WaitGroup) error {
	defer wg.Done()
	wg.Add(1)
	// Step 1: Count total entries in the table
	countQuery := "SELECT COUNT(*) FROM " + tableName + " WHERE status = 'done'"
	var totalEntries int
	if err := db.QueryRow(countQuery).Scan(&totalEntries); err != nil {
		return fmt.Errorf("failed to count entries in table %s: %w", tableName, err)
	}
	if verbose || debug {
		log.Printf("DeleteMaxEntriesHistory - Table %s has %d entries\n", tableName, totalEntries)
	}

	// Step 2: If total entries are within the limit, return
	if totalEntries <= maxEntries {
		if debug {
			log.Printf("DeleteMaxEntriesHistory - No deletion needed; %d <= %d\n", totalEntries, maxEntries)
		}
		return nil
	}

	// Step 3: Calculate the number of entries to delete
	entriesToDelete := totalEntries - maxEntries
	if debug {
		log.Printf("DeleteMaxEntriesHistory - Need to delete %d entries from table %s\n", entriesToDelete, tableName)
	}

	// Step 4: Delete the oldest entries
	// Assuming the table has columns `id` and `created_at`. Adjust as needed.
	deleteQuery := fmt.Sprintf(`
	WITH cte AS (
		SELECT ID
		FROM %s
		WHERE status = "done"
		ORDER BY createdAt ASC
		LIMIT ?
	)
	DELETE FROM %s
	WHERE ID IN (SELECT ID FROM cte)
`, tableName, tableName)

	result, err := execWithRetry(db, wg, deleteQuery, entriesToDelete)

	if err != nil {
		return fmt.Errorf("DeleteMaxEntriesHistory - failed to delete old entries from table %s: %w", tableName, err)
	}

	if debug {
		log.Println("DeleteMaxEntriesHistory - execWithRetry", result, err)
	}

	// Step 5: Log the number of rows affected
	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteMaxEntriesHistory - failed to fetch rows affected: %w", err)
	}
	if verbose || debug {
		log.Printf("DeleteMaxEntriesHistory - Deleted %d old entries from table %s\n", rowsDeleted, tableName)
	}

	return nil
}
