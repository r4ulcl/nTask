// manager.go
// Package database provides functions for managing database connections and executing SQL statements.

package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

var sqlInit = `
CREATE TABLE IF NOT EXISTS worker (
    name VARCHAR(255) PRIMARY KEY,
    DefaultThreads INT,
    IddleThreads INT,
    up BOOLEAN,
    downCount INT
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
	// Create a connection string.
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, database)

	if debug {
		log.Println("DB ConnectDB - dataSourceName", dataSourceName)
	}
	// Open a new connection to the MySQL database.
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Check if the connection is successful.
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Initialize the database structure from SQL file.
	err = initFromVar(db, verbose, debug)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

// initFromFile initializes the database structure by executing SQL statements from a file.
// It takes a pointer to the sql.DB object and the file path as input.
// It returns an error if the initialization fails.
func initFromVar(db *sql.DB, verbose, debug bool) error {
	// Split the content of the SQL file into individual statements
	sqlStatements := strings.Split(string(sqlInit), ";")

	if verbose || debug {
		log.Println("initFromVar sqlInit")
	}

	// Execute each SQL statement
	for _, statement := range sqlStatements {
		// Trim leading and trailing whitespaces
		sqlStatement := strings.TrimSpace(statement)

		// Skip empty statements
		if sqlStatement == "" {
			continue
		}

		// Execute the SQL statement
		_, err := db.Query(sqlStatement)
		if err != nil {
			return err
		}
	}

	return nil
}
