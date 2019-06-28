package repo

import (
	"database/sql"

	cError "sorcia/error"
)

// CreateRepo ...
func CreateRepo(db *sql.DB) {
	_, err := db.Query("CREATE TABLE IF NOT EXISTS repository (id BIGSERIAL, repo_from INTEGER REFERENCES account(id), name VARCHAR(255) UNIQUE NOT NULL, description VARCHAR(500), type VARCHAR(255) NOT NULL, PRIMARY KEY (id, repo_from))")
	cError.CheckError(err)
}

// CreateRepoStruct struct
type CreateRepoStruct struct {
	Name string
	Description string
	RepoType string
	UserID int
}

// InsertRepo ...
func InsertRepo(db *sql.DB, crs CreateRepoStruct) {
	var lastInsertID int

	err := db.QueryRow("INSERT INTO repository (repo_from, name, description, type) VALUES ($1, $2, $3, $4) returning id", crs.UserID, crs.Name, crs.Description, crs.RepoType).Scan(&lastInsertID)
	cError.CheckError(err)
}