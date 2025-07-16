// manager.go
// Package database provides functions for managing database connections and executing SQL statements.

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

const (
	maxRetries                = 3
	initialBackOff            = 50 * time.Millisecond
	defaultSelectLimit        = 1000
	defaultHistoryLimit       = 5000
	maxConcurrentGeneralDBOps = 1
	maxConcurrentInsertDBOps  = 10
	dbConnMaxLifetime         = 30 * time.Minute
	dbMaxOpenConns            = 50
	dbMaxIdleConns            = 25
)

// semaphore to throttle concurrent calls to execWithRetry
var (
	insertSemaphore  = make(chan struct{}, maxConcurrentInsertDBOps)
	generalSemaphore = make(chan struct{}, maxConcurrentGeneralDBOps)
)

func init() {
	// Seed the RNG for backoff jitter
	rand.Seed(time.Now().UnixNano())
}

var sqlInit = `
CREATE TABLE IF NOT EXISTS worker (
    name VARCHAR(255) PRIMARY KEY,
    DefaultThreads INT,
    IddleThreads INT,
    up BOOLEAN,
    downCount INT,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task (
    ID VARCHAR(255) PRIMARY KEY,
    notes LONGTEXT,
    commands LONGTEXT,
    files LONGTEXT,
    name TEXT,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    executedAt TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    status VARCHAR(255),
    duration INT DEFAULT 0,
    workerName VARCHAR(255),
    username VARCHAR(255),
    priority INT DEFAULT 0,
    timeout INT DEFAULT 0,
    callbackURL TEXT,
    callbackToken TEXT,
    INDEX idx_status (status)
);
`

// ConnectDB creates a new Manager instance and initializes the database connection.
// It takes the username, password, host, port, and database name as input.
// It returns a pointer to the sql.DB object and an error if the connection fails.
func ConnectDB(username, password, host, port, database string, verbose, debug bool) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&clientFoundRows=true",
		username,
		password,
		host,
		port,
		database,
	)
	if debug {
		log.Println("DB ConnectDB - DSN:", dsn)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// pool sizing
	db.SetMaxOpenConns(dbMaxOpenConns)
	db.SetMaxIdleConns(dbMaxIdleConns)
	db.SetConnMaxLifetime(dbConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := initFromVar(db, verbose, debug); err != nil {
		return nil, err
	}
	return db, nil
}

func initFromVar(db *sql.DB, verbose, debug bool) error {
	stmts := strings.Split(sqlInit, ";")
	if verbose || debug {
		log.Println("initFromVar: applying schema")
	}
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("initFromVar executing %q: %w", s, err)
		}
	}
	return nil
}

// execWithRetry wraps db.Exec to retry on MySQL deadlock (Error 1213).
func execWithRetry(db *sql.DB, isInsert bool, query string, args ...interface{}) (sql.Result, error) {

	// pick the semaphore based on the isInsert flag
	dbSemaphore := generalSemaphore
	if isInsert {
		dbSemaphore = insertSemaphore
	}

	dbSemaphore <- struct{}{}
	defer func() { <-dbSemaphore }()

	var err error
	backOff := initialBackOff
	for i := 0; i < maxRetries; i++ {
		var res sql.Result
		// Attach a short context so the query does not block forever if the connection freezes.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		res, err = db.ExecContext(ctx, query, args...)
		cancel()
		if err == nil {
			return res, nil
		}
		if merr, ok := err.(*mysql.MySQLError); ok && merr.Number == 1213 { // deadlock found
			time.Sleep(backOff)
			backOff *= 2
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("deadlock after %d retries for query %q: %w", maxRetries, query, err)
}

// serializeToJSON marshals a slice into a JSON string.
func serializeToJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// prepareTaskQuery prepare task insertion or update in the database.
func prepareTaskQuery(task globalstructs.Task, verbose, debug bool) (commandJSON, filesJSON string, err error) {
	commandJSON, err = serializeToJSON(task.Commands)
	if err != nil {
		return "", "", err
	}
	filesJSON, err = serializeToJSON(task.Files)
	if err != nil {
		return "", "", err
	}
	if verbose || debug {
		log.Printf("prepareTaskQuery: filesJSON=%s commandJSON=%s", filesJSON, commandJSON)
	}
	return
}
