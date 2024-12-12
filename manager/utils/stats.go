package utils

import (
	"database/sql"

	"github.com/r4ulcl/nTask/manager/database"
)

// GetStatusTask function to get task status, pending, running, etc
func GetStatusTask(db *sql.DB, verbose, debug bool) (StatusTask, error) {
	task := StatusTask{
		Pending: 0,
		Running: 0,
		Done:    0,
		Failed:  0,
		Deleted: 0,
	}

	pending, err := database.GetPendingCount(db, verbose, debug)
	if err != nil {
		return task, err
	}
	task.Pending = pending

	running, err := database.GetRunningCount(db, verbose, debug)
	if err != nil {
		return task, err
	}
	task.Running = running

	done, err := database.GetDoneCount(db, verbose, debug)
	if err != nil {
		return task, err
	}
	task.Done = done

	failed, err := database.GetFailedCount(db, verbose, debug)
	if err != nil {
		return task, err
	}
	task.Failed = failed

	deleted, err := database.GetDeletedCount(db, verbose, debug)
	if err != nil {
		return task, err
	}
	task.Deleted = deleted

	return task, nil
}

// GetStatusWorker func to get status up, down of workers
func GetStatusWorker(db *sql.DB, verbose, debug bool) (StatusWorker, error) {
	worker := StatusWorker{
		Up:   0,
		Down: 0,
	}

	up, err := database.GetUpCount(db, verbose, debug)
	if err != nil {
		return worker, err
	}
	worker.Up = up

	down, err := database.GetDownCount(db, verbose, debug)
	if err != nil {
		return worker, err
	}
	worker.Down = down
	return worker, nil
}
