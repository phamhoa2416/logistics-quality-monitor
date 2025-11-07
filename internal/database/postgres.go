package database

import (
	"database/sql"
	"fmt"
	"log"
	"logistics-quality-monitor/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Data connection established")

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func (d *Database) Health() error {
	return d.DB.Ping()
}
