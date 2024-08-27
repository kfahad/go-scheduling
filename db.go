package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func setupDatabase() (*sqlx.DB, error) {
	connStr := "user=postgres password=go_sche dbname=jobs sslmode=disable"
	db, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Create jobs table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			payload TEXT,
			dependencies TEXT[],
			retry_count INT,
			max_retries INT,
			status TEXT,
			priority INT
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}
