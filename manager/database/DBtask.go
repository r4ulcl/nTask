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

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// AddTask adds a task to the database.
func AddTask(db *sql.DB, task globalstructs.Task, verbose, debug bool) error {
	const q = `INSERT INTO task
        (ID, notes, commands, files, name, status, duration, WorkerName, username, priority, timeout, callbackURL, callbackToken)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	cmdJSON, fileJSON, err := prepareTaskQuery(task, verbose, debug)
	if err != nil {
		return err
	}
	if _, err = execWithRetry(db, true, q,
		task.ID, task.Notes, cmdJSON, fileJSON, task.Name, task.Status,
		task.Duration, task.WorkerName, task.Username, task.Priority,
		task.Timeout, task.CallbackURL, task.CallbackToken); err != nil {
		return fmt.Errorf("AddTask: %w", err)
	}
	return nil
}

// UpdateTask updates all fields of a task in the database.
func UpdateTask(db *sql.DB, task globalstructs.Task, verbose, debug bool) error {
	if debug {
		log.Println("UpdateTask1:", task)
	}

	const q = `UPDATE task SET
            notes=?, commands=?, files=?, name=?, status=?, duration=?,
            WorkerName=?, priority=?, timeout=?, callbackURL=?, callbackToken=?, updatedAt = NOW()
        WHERE ID=?`

	cmdJSON, fileJSON, err := prepareTaskQuery(task, verbose, debug)
	if err != nil {
		return err
	}

	res, err := execWithRetry(db, false, q,
		task.Notes, cmdJSON, fileJSON, task.Name, task.Status, task.Duration,
		task.WorkerName, task.Priority, task.Timeout, task.CallbackURL,
		task.CallbackToken, task.ID,
	)
	if err != nil {
		return fmt.Errorf("UpdateTask error: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("UpdateTask: no task with ID %s found (possible race)", task.ID)
	}

	if debug {
		log.Println("UpdateTask2: ", task)
	}
	return nil
}

// RmTask deletes a task from the database.
func RmTask(db *sql.DB, id string, verbose, debug bool) error {
	const q = `DELETE FROM task WHERE ID = ?`
	res, err := execWithRetry(db, false, q, false, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("RmTask: task %s not found", id)
	}
	return nil
}

// buildFiltersWithParams constructs SQL filters using query parameters safely.
func buildFiltersWithParams(query url.Values) (string, []interface{}) {
	var (
		filters []string
		args    []interface{}
	)
	add := func(key, cond string) {
		if v := query.Get(key); v != "" {
			filters = append(filters, cond)
			args = append(args, v)
		}
	}
	add("ID", "ID LIKE ?")
	add("notes", "notes LIKE ?")
	add("commands", "commands LIKE ?")
	add("files", "files LIKE ?")
	add("name", "name LIKE ?")
	add("createdAt", "createdAt LIKE ?")
	add("updatedAt", "updatedAt LIKE ?")
	add("executedAt", "executedAt LIKE ?")
	add("status", "status = ?")
	add("duration", "duration = ?")
	add("workerName", "workerName LIKE ?")
	add("username", "username LIKE ?")
	add("priority", "priority = ?")
	add("timeout", "timeout = ?")
	add("callbackURL", "callbackURL = ?")
	add("callbackToken", "callbackToken = ?")
	return strings.Join(filters, " AND "), args
}

func buildOrderByAndLimit(page, limit int) (string, int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultSelectLimit
	}
	offset := (page - 1) * limit
	return " ORDER BY priority DESC, createdAt ASC", limit, offset
}

// GetTasks retrieves tasks from the database using URL parameters as filters.
func GetTasks(r *http.Request, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	queryParams := r.URL.Query()
	filters, args := buildFiltersWithParams(queryParams)
	orderBy, limit, offset := buildOrderByAndLimit(getInt(queryParams, "page", 1), getInt(queryParams, "limit", defaultSelectLimit))

	sqlStr := "SELECT ID, notes, commands, files, name, createdAt, updatedAt, executedAt, status, duration, WorkerName, username, priority, timeout, callbackURL, callbackToken FROM task WHERE 1=1"
	if filters != "" {
		sqlStr += " AND " + filters
	}
	sqlStr += orderBy + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if debug {
		log.Println("GetTasks SQL:", sqlStr)
		log.Println("Args:", args)
	}
	return getTasksSQL(sqlStr, args, db, verbose, debug)
}

func getInt(v url.Values, key string, d int) int {
	if s := v.Get(key); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return d
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
	if limit <= 0 {
		limit = 1
	}
	const q = `SELECT ID, notes, commands, files, name, createdAt, updatedAt, executedAt, status, duration, WorkerName, username, priority, timeout, callbackURL, callbackToken
               FROM task WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC LIMIT ?`
	return getTasksSQL(q, []interface{}{limit}, db, verbose, debug)
}

// getTasksSQL executes a parameterized SQL query to fetch tasks.
func getTasksSQL(sqlQuery string, args []interface{}, db *sql.DB, verbose, debug bool) ([]globalstructs.Task, error) {
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		if debug {
			log.Println("getTasksSQL query error:", err)
		}
		return nil, err
	}
	defer rows.Close()

	var tasks []globalstructs.Task
	for rows.Next() {
		var (
			t           globalstructs.Task
			commandsStr string
			filesStr    string
		)
		if err = rows.Scan(&t.ID, &t.Notes, &commandsStr, &filesStr, &t.Name,
			&t.CreatedAt, &t.UpdatedAt, &t.ExecutedAt, &t.Status, &t.Duration,
			&t.WorkerName, &t.Username, &t.Priority, &t.Timeout, &t.CallbackURL, &t.CallbackToken); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(commandsStr), &t.Commands); err != nil {
			return nil, fmt.Errorf("parse commands: %w", err)
		}
		if err = json.Unmarshal([]byte(filesStr), &t.Files); err != nil {
			return nil, fmt.Errorf("parse files: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTask gets task filtered by id
func GetTask(db *sql.DB, id string, verbose, debug bool) (globalstructs.Task, error) {
	const q = `SELECT ID, notes, createdAt, updatedAt, executedAt, commands, files, name, status, duration, WorkerName,
                      username, priority, timeout, callbackURL, callbackToken
               FROM task WHERE ID = ?`
	var (
		t           globalstructs.Task
		commandsStr string
		filesStr    string
	)
	err := db.QueryRow(q, id).Scan(&t.ID, &t.Notes, &t.CreatedAt, &t.UpdatedAt, &t.ExecutedAt, &commandsStr, &filesStr,
		&t.Name, &t.Status, &t.Duration, &t.WorkerName, &t.Username, &t.Priority, &t.Timeout, &t.CallbackURL, &t.CallbackToken)
	if err != nil {
		return t, err
	}
	if err = json.Unmarshal([]byte(commandsStr), &t.Commands); err != nil {
		return t, fmt.Errorf("parse commands: %w", err)
	}
	if err = json.Unmarshal([]byte(filesStr), &t.Files); err != nil {
		return t, fmt.Errorf("parse files: %w", err)
	}
	return t, nil
}

// Generic helper function to execute a database update
func executeDBUpdate(db *sql.DB, query string, args []interface{}, verbose, debug bool, taskName string) error {
	_, err := execWithRetry(db, false, query, args...)

	if err != nil {
		if debug || verbose {
			log.Printf("DB Error %s: %v", taskName, err)
		}
		return err
	}
	return nil
}

// SetTasksWorkerPending Function to set tasks worker status to 'pending'
func SetTasksWorkerPending(db *sql.DB, workerName string, verbose, debug bool) error {
	query := "UPDATE task SET status = 'pending', updatedAt = NOW() WHERE workerName = ? AND status = 'running'"
	args := []interface{}{workerName}
	return executeDBUpdate(db, query, args, verbose, debug, "DBTask: SetTasksWorkerPending")
}

// SetTaskExecutedAtNow Function to set task's executedAt timestamp to now()
func SetTaskExecutedAtNow(db *sql.DB, id string, verbose, debug bool) error {
	res, err := execWithRetry(db, false, "UPDATE task SET executedAt = NOW(), updatedAt = NOW() WHERE ID = ?", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SetTaskExecutedAtNow: task %s not found", id)
	}
	return nil
}

// SetTaskWorkerName saves the worker name of the task in the database
func SetTaskWorkerName(db *sql.DB, id, workerName string, verbose, debug bool) error {
	res, err := execWithRetry(db, false, "UPDATE task SET WorkerName = ?, updatedAt = NOW() WHERE ID = ?", workerName, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SetTaskWorkerName: task %s not found", id)
	}
	return nil
}

func ClearWorkerName(db *sql.DB, workerName string, verbose, debug bool) error {
	_, err := execWithRetry(db, false, "UPDATE task SET WorkerName = '', updatedAt = NOW() WHERE WorkerName = ?", workerName)
	return err
}

func SetTaskStatus(db *sql.DB, id, status string, verbose, debug bool) error {
	res, err := execWithRetry(db, false, "UPDATE task SET status = ?, updatedAt = NOW() WHERE ID = ?", status, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SetTaskStatus: task %s not found", id)
	}
	if debug {
		log.Println("SetTaskStatus", id, status)
	}
	return nil
}

func TransitionTasks(db *sql.DB, fromStatus, toStatus string, verbose, debug bool) (int64, error) {
	res, err := execWithRetry(db, false, "UPDATE task SET status = ?, updatedAt = NOW() WHERE status = ?", toStatus, fromStatus)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ---------------- Statistics & housekeeping ------------------------------

func GetCountByStatus(status string, db *sql.DB, verbose, debug bool) (int, error) {
	const q = `SELECT COUNT(*) FROM task WHERE status = ?`
	var c int
	if err := db.QueryRow(q, status).Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// DeleteMaxEntriesHistory keeps only the newest <maxEntries> rows whose status = 'done'.
func DeleteMaxEntriesHistory(db *sql.DB, maxEntries int, table string, verbose, debug bool) error {
	if maxEntries <= 0 {
		maxEntries = defaultHistoryLimit
	}
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE status = 'done'").Scan(&total); err != nil {
		return err
	}
	if total <= maxEntries {
		return nil // nothing to do
	}
	del := total - maxEntries
	cte := fmt.Sprintf(`WITH old AS (
            SELECT ID FROM %s WHERE status = 'done' ORDER BY createdAt ASC LIMIT ?)
        DELETE FROM %s WHERE ID IN (SELECT ID FROM old)`, table, table)
	if _, err := execWithRetry(db, false, cte, del); err != nil {
		return err
	}
	if verbose || debug {
		log.Printf("DeleteMaxEntriesHistory: trimmed %d → %d rows in %s", total, maxEntries, table)
	}
	return nil
}

// setTasksWorkerEmpty remove the worker name of the task in the database
func setTasksWorkerEmpty(db *sql.DB, workerName string, verbose, debug bool) error {

	// Update the workerName column of the task table for the given ID
	query := "UPDATE task SET workerName = '', updatedAt = NOW() WHERE  workerName = ?"
	_, err := execWithRetry(db, false, query, workerName)
	if err != nil {
		if debug || verbose {
			log.Println("DB Error DBTask SetTaskWorkerName: ", err)
		}
		return err
	}
	return nil
}

// SetTasksStatusIfStatus saves the status of the task in the database if current status is currentStatus
func SetTasksStatusIfStatus(currentStatus string, db *sql.DB, newStatus string, verbose, debug bool) error {

	// Update the status column of the task table for the given ID
	query := "UPDATE task SET status = ?, updatedAt = NOW() WHERE status = ?"
	_, err := execWithRetry(db, false, query, newStatus, currentStatus)
	if err != nil {
		if debug || verbose {
			log.Println("DB Error DBTask SetTasksStatusIfRunning: ", err)
		}
		return err
	}
	return nil
}

// SetTaskExecutedAt saves current time as executedAt
func SetTaskExecutedAt(executedAt string, db *sql.DB, id string, verbose, debug bool) error {

	// Update the status column of the task table for the given ID
	query := "UPDATE task SET executedAt = ?, updatedAt = NOW() WHERE status = ?"
	_, err := execWithRetry(db, false, query, executedAt, id)
	if err != nil {
		if debug {
			log.Println("DB Error DBTask SetTaskExecutedAt: ", err)
		}
		return err
	}
	return nil
}

// Count
