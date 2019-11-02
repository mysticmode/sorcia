package config

import (
	"database/sql"

	// This import is here because we have a separate config file.
	_ "github.com/lib/pq"
)

// Env struct
type Env struct {
	DB *sql.DB
}

// NewDB ...
func NewDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
