package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ProcessingLog struct {
	ID          int64
	SourceFile  string
	OutputFile  string
	R2URL       string
	ProcessedAt time.Time
	Status      string
	Error       string
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS processing_logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        source_file TEXT NOT NULL,
        output_file TEXT NOT NULL,
        r2_url TEXT,
        processed_at TIMESTAMP NOT NULL,
        status TEXT NOT NULL,
        error TEXT
    );`

	_, err = db.Exec(createTableSQL)
	return db, err
}

func LogProcess(db *sql.DB, log ProcessingLog) error {
	_, err := db.Exec(`
        INSERT INTO processing_logs 
        (source_file, output_file, r2_url, processed_at, status, error)
        VALUES (?, ?, ?, ?, ?, ?)`,
		log.SourceFile, log.OutputFile, log.R2URL, log.ProcessedAt, log.Status, log.Error,
	)
	return err
}
