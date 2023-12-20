// manager.go
// Package database provides functions for managing database connections and executing SQL statements.

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectDB creates a new Manager instance and initializes the database connection.
// It takes the username, password, host, port, and database name as input.
// It returns a pointer to the sql.DB object and an error if the connection fails.
func ConnectDB(username, password, host, port, database string, verbose, debug bool) (*sql.DB, error) {
	// Create a connection string.
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, database)

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
	sqlFile := "sql.sql"
	err = initFromFile(db, sqlFile, verbose, debug)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

// initFromFile initializes the database structure by executing SQL statements from a file.
// It takes a pointer to the sql.DB object and the file path as input.
// It returns an error if the initialization fails.
func initFromFile(db *sql.DB, filePath string, verbose, debug bool) error {
	// Read the SQL file
	sqlFile, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Split the content of the SQL file into individual statements
	sqlStatements := strings.Split(string(sqlFile), ";")

	// Execute each SQL statement
	for _, statement := range sqlStatements {
		// Trim leading and trailing whitespaces
		sqlStatement := strings.TrimSpace(statement)

		// Skip empty statements
		if sqlStatement == "" {
			continue
		}

		// Execute the SQL statement
		_, err := db.Exec(sqlStatement)
		if err != nil {
			return err
		}
	}

	return nil
}
