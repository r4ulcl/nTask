package database

import (
	"database/sql"
	"fmt"
	"log"

	globalstructs "github.com/r4ulcl/nTask/globalstructs"
)

// -------------------------------------------------------------------------
// Inserts / deletes
// -------------------------------------------------------------------------

// AddWorker inserts a new worker row.
func AddWorker(db *sql.DB, w *globalstructs.Worker, verbose, debug bool) error {
	const q = `INSERT INTO worker (name, defaultThreads, iddleThreads, up, downCount, updatedAt)
	           VALUES (?, ?, ?, ?, ?, NOW())`
	if _, err := execWithRetry(db, true, q, w.Name, w.DefaultThreads, w.IddleThreads, w.UP, w.DownCount); err != nil {
		return fmt.Errorf("AddWorker: %w", err)
	}
	return nil
}

// RmWorkerName deletes a worker by name and detaches its running tasks.
func RmWorkerName(db *sql.DB, name string, verbose, debug bool) error {
	const del = `DELETE FROM worker WHERE name = ?`
	res, err := execWithRetry(db, false, del, name)
	if err != nil {
		return fmt.Errorf("RmWorkerName: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("RmWorkerName: worker %s not found", name)
	}
	// orphan tasks → pending
	if err := SetTasksWorkerPending(db, name, verbose, debug); err != nil {
		return err
	}
	if err := clearWorkerName(db, name, verbose, debug); err != nil {
		return err
	}
	return nil
}

// -------------------------------------------------------------------------
// Select helpers
// -------------------------------------------------------------------------

const workerSelectCols = `name, defaultThreads, iddleThreads, up, downCount, updatedAt`

// GetWorkers returns every row in the worker table.
func GetWorkers(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	q := "SELECT " + workerSelectCols + " FROM worker"
	return getWorkerSQL(q, db, verbose, debug)
}

// GetWorker fetches a single worker by name.
func GetWorker(db *sql.DB, name string, verbose, debug bool) (globalstructs.Worker, error) {
	q := "SELECT " + workerSelectCols + " FROM worker WHERE name = ?"
	rows, err := getWorkerSQL(q, db, verbose, debug, name)
	if err != nil {
		return globalstructs.Worker{}, err
	}
	if len(rows) == 0 {
		return globalstructs.Worker{}, sql.ErrNoRows
	}
	return rows[0], nil
}

// GetWorkerIddle returns workers that are up and have spare threads.
func GetWorkerIddle(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	q := "SELECT " + workerSelectCols + " FROM worker WHERE up = TRUE AND iddleThreads > 0 ORDER BY RAND()"
	return getWorkerSQL(q, db, verbose, debug)
}

// GetWorkerUP returns all workers with up = true.
func GetWorkerUP(db *sql.DB, verbose, debug bool) ([]globalstructs.Worker, error) {
	q := "SELECT " + workerSelectCols + " FROM worker WHERE up = TRUE"
	return getWorkerSQL(q, db, verbose, debug)
}

// -------------------------------------------------------------------------
// Updates
// -------------------------------------------------------------------------

// UpdateWorker replaces every mutable column of the given worker.
func UpdateWorker(db *sql.DB, w *globalstructs.Worker, verbose, debug bool) error {
	const q = `UPDATE worker SET defaultThreads = ?, iddleThreads = ?, up = ?, downCount = ?, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, w.DefaultThreads, w.IddleThreads, w.UP, w.DownCount, w.Name)
	if err != nil {
		return fmt.Errorf("UpdateWorker: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("UpdateWorker: worker %s not found", w.Name)
	}
	return nil
}

// SetWorkerUPto toggles the up column.
func SetWorkerUPto(db *sql.DB, name string, up bool, verbose, debug bool) error {
	const q = `UPDATE worker SET up = ?, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, up, name)
	if err != nil {
		return fmt.Errorf("SetWorkerUPto: %w", err)
	}

	// RowsAffected==0 could mean “no matching row” OR “already in desired state”
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("SetWorkerUPto (RowsAffected): %w", err)
	}
	if n == 0 {
		// Check existence explicitly
		var dummy int
		err := db.QueryRow("SELECT 1 FROM worker WHERE name = ?", name).Scan(&dummy)
		if err == sql.ErrNoRows {
			return fmt.Errorf("SetWorkerUPto: worker %s not found", name)
		} else if err != nil {
			return fmt.Errorf("SetWorkerUPto (existence check): %w", err)
		}
		// row exists but was already up/down as requested → treat as success
	}

	if debug {
		log.Printf("SetWorkerUPto: worker %s up set to %t", name, up)
	}
	return nil
}

// SetIddleThreadsTo sets the iddleThreads value.
func SetIddleThreadsTo(db *sql.DB, name string, idle int, verbose, debug bool) error {
	const q = `UPDATE worker SET iddleThreads = ?, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, idle, name)
	if err != nil {
		return fmt.Errorf("SetIddleThreadsTo: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SetIddleThreadsTo: worker %s not found", name)
	}
	return nil
}

// SubtractWorkerIddleThreads1 decrements iddleThreads by 1 if > 0.
func SubtractWorkerIddleThreads1(db *sql.DB, name string, verbose, debug bool) error {
	const q = `UPDATE worker SET iddleThreads = CASE WHEN iddleThreads > 0 THEN iddleThreads - 1 ELSE 0 END, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, name)
	if err != nil {
		return fmt.Errorf("SubtractWorkerIddleThreads1: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SubtractWorkerIddleThreads1: worker %s not found", name)
	}
	return nil
}

// Down‑count helpers -------------------------------------------------------

func GetWorkerDownCount(db *sql.DB, name string, verbose, debug bool) (int, error) {
	var dc int
	if err := db.QueryRow("SELECT downCount FROM worker WHERE name = ?", name).Scan(&dc); err != nil {
		return 0, err
	}
	return dc, nil
}

func SetWorkerDownCount(db *sql.DB, name string, count int, verbose, debug bool) error {
	const q = `UPDATE worker SET downCount = ?, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, count, name)
	if err != nil {
		return fmt.Errorf("SetWorkerDownCount: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("SetWorkerDownCount: worker %s not found", name)
	}
	return nil
}

func AddWorkerDownCount(db *sql.DB, name string, verbose, debug bool) error {
	const q = `UPDATE worker SET downCount = downCount + 1, updatedAt = NOW() WHERE name = ?`
	res, err := execWithRetry(db, false, q, name)
	if err != nil {
		return fmt.Errorf("AddWorkerDownCount: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("AddWorkerDownCount: worker %s not found", name)
	}
	return nil
}

// -------------------------------------------------------------------------
// Simple aggregates
// -------------------------------------------------------------------------

func GetUpCount(db *sql.DB, verbose, debug bool) (int, error) {
	return getBoolCount(db, true)
}

func GetDownCount(db *sql.DB, verbose, debug bool) (int, error) {
	return getBoolCount(db, false)
}

func getBoolCount(db *sql.DB, up bool) (int, error) {
	query := "SELECT COUNT(*) FROM worker WHERE up = ?"
	var c int
	if err := db.QueryRow(query, up).Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// -------------------------------------------------------------------------
// Row‑mapper
// -------------------------------------------------------------------------

func getWorkerSQL(sqlStr string, db *sql.DB, verbose, debug bool, args ...interface{}) ([]globalstructs.Worker, error) {
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		if debug {
			log.Println("getWorkerSQL query error:", err)
		}
		return nil, err
	}
	defer rows.Close()

	var workers []globalstructs.Worker
	for rows.Next() {
		var w globalstructs.Worker
		if err = rows.Scan(&w.Name, &w.DefaultThreads, &w.IddleThreads, &w.UP, &w.DownCount, &w.UpdatedAt); err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, rows.Err()
}
