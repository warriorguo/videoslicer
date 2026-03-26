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
	migrations := []string{
		`ALTER TABLE video_tasks ALTER COLUMN frame_interval_sec TYPE DOUBLE PRECISION`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			// Ignore if already the correct type
			log.Printf("Migration skipped (may already be applied): %v", err)
		}
	}
	return nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}
