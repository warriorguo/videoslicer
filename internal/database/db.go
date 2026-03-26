package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDB(config Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s",
		config.Host, config.Port, config.DBName, config.User, config.SSLMode)
	if config.Password != "" {
		dsn += fmt.Sprintf(" password=%s", config.Password)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) EnsureSchema() error {
	// Check current column type
	var dataType string
	err := db.QueryRow(
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'video_tasks' AND column_name = 'frame_interval_sec'`,
	).Scan(&dataType)
	if err != nil {
		log.Printf("EnsureSchema: failed to query column type: %v", err)
		return nil
	}
	log.Printf("EnsureSchema: frame_interval_sec current type = %s", dataType)

	if dataType == "double precision" {
		log.Printf("EnsureSchema: column already DOUBLE PRECISION, no migration needed")
		return nil
	}

	_, err = db.Exec(`ALTER TABLE video_tasks ALTER COLUMN frame_interval_sec TYPE DOUBLE PRECISION USING frame_interval_sec::double precision`)
	if err != nil {
		log.Printf("EnsureSchema: ALTER TABLE failed: %v", err)
		return fmt.Errorf("migration failed: %w", err)
	}
	log.Printf("EnsureSchema: successfully migrated frame_interval_sec to DOUBLE PRECISION")
	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
