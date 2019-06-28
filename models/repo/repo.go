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
	Name        string
	Description string
	RepoType    string
	UserID      int
}

// InsertRepo ...
func InsertRepo(db *sql.DB, crs CreateRepoStruct) {
	var lastInsertID int

	err := db.QueryRow("INSERT INTO repository (repo_from, name, description, type) VALUES ($1, $2, $3, $4) returning id", crs.UserID, crs.Name, crs.Description, crs.RepoType).Scan(&lastInsertID)
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
	Type        string
}

// GetReposFromUserID ...
func GetReposFromUserID(db *sql.DB, userID int) *GetReposFromUserIDResponse {
	rows, err := db.Query("SELECT name, description, type FROM repository WHERE repo_from = $1", userID)
	cError.CheckError(err)

	var grfur GetReposFromUserIDResponse
	var rds ReposDetailStruct

	for rows.Next() {
		err := rows.Scan(&rds.Name, &rds.Description, &rds.Type)
		cError.CheckError(err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return &grfur
}
