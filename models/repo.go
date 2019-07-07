package models

import (
	"database/sql"

	cError "sorcia/error"
)

// CreateRepo ...
func CreateRepo(db *sql.DB) {
	_, err := db.Query("CREATE TABLE IF NOT EXISTS repository (id BIGSERIAL, repo_from INTEGER REFERENCES account(id), name VARCHAR(255) UNIQUE NOT NULL, description VARCHAR(500), is_private BOOLEAN NOT NULL DEFAULT FALSE, PRIMARY KEY (id, repo_from))")
	cError.CheckError(err)
}

// CreateRepoStruct struct
type CreateRepoStruct struct {
	Name        string
	Description string
	IsPrivate   int
	UserID      int
}

// InsertRepo ...
func InsertRepo(db *sql.DB, crs CreateRepoStruct) {
	var lastInsertID int

	err := db.QueryRow("INSERT INTO repository (repo_from, name, description, is_private) VALUES ($1, $2, $3, $4) returning id", crs.UserID, crs.Name, crs.Description, crs.IsPrivate).Scan(&lastInsertID)
	cError.CheckError(err)
}

// GetReposFromUserIDResponse struct
type GetReposFromUserIDResponse struct {
	Repositories []ReposDetailStruct
}

// ReposDetailStruct struct
type ReposDetailStruct struct {
	Name        string
	Description string
	IsPrivate   string
}

// GetReposFromUserID ...
func GetReposFromUserID(db *sql.DB, userID int) *GetReposFromUserIDResponse {
	rows, err := db.Query("SELECT name, description, is_private FROM repository WHERE repo_from = $1", userID)
	cError.CheckError(err)

	var grfur GetReposFromUserIDResponse
	var rds ReposDetailStruct

	for rows.Next() {
		err := rows.Scan(&rds.Name, &rds.Description, &rds.IsPrivate)
		cError.CheckError(err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return &grfur
}
