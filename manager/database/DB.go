// manager.go
package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectDB creates a new Manager instance and initializes the database connection.
func ConnectDB(username, password, host, port, database string) (*sql.DB, error) {
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

	sqlFile := "manager/database/sql.sql"
	err = initFromFile(db, sqlFile)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

// init database from file filePath
func initFromFile(db *sql.DB, filePath string) error {
	// Read the SQL file
	sqlFile, err := ioutil.ReadFile(filePath)
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
