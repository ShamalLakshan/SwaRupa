package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

const dbFile = "albumart.db"
const migrationFile = "migrations/0001_initial.sql"

// Connect opens the SQLite DB (local) and applies migration if first run
func Connect() *sql.DB {
	firstRun := false
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		firstRun = true
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	if firstRun {
		fmt.Println("Initializing database schema...")
		schema, err := ioutil.ReadFile(migrationFile)
		if err != nil {
			log.Fatalf("Failed to read migration file: %v", err)
		}

		_, err = db.Exec(string(schema))
		if err != nil {
			log.Fatalf("Failed to execute migration: %v", err)
		}

		fmt.Println("Database initialized successfully!")
	}

	return db
}
