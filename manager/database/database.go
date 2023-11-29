// manager.go
package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

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

	err = InitDB(db)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

func InitDB(db *sql.DB) error {

	b, err := ioutil.ReadFile("manager/database/sql.sql") // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	databaseSQL := string(b) // convert content to a 'string'

	//Init db
	_, err = db.Exec(databaseSQL)

	if err != nil {
		return err
	}
	return nil
}
