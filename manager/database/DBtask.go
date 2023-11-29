package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/r4ulcl/NetTask/manager/utils"
)

func AddTask(db *sql.DB, task utils.Task) error {
	// Marshal the Worker struct to JSON
	jsonData, err := json.Marshal(task)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Insert the JSON data into the MySQL table
	_, err = db.Exec("INSERT INTO task (ID, data) VALUES (?, ?)",
		task.ID, jsonData)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func RmTask(db *sql.DB, ID string) error {
	fmt.Println("TODO RmTask")
	return nil
}

func StopTask(db *sql.DB, task utils.Task) error {
	fmt.Println("TODO StopTask")
	return nil
}

func GetTasks(db *sql.DB) ([]utils.Task, error) {
	var tasks []utils.Task

	fmt.Println("TODO GetTasks")
	return tasks, nil
}

func GetTask(db *sql.DB, ID string) (utils.Task, error) {
	var task utils.Task

	fmt.Println("TODO GetTask")
	return task, nil
}
