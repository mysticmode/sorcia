package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateRepo ...
func CreateRepo(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS repository (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, name TEXT UNIQUE NOT NULL, description TEXT, is_private BOOLEAN DEFAULT 0, FOREIGN KEY (user_id) REFERENCES account(id))")
	errorhandler.CheckError(err)

	_, err = stmt.Exec()
	errorhandler.CheckError(err)
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
	stmt, err := db.Prepare("INSERT INTO repository (user_id, name, description, is_private) VALUES (?, ?, ?, ?)")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(crs.UserID, crs.Name, crs.Description, crs.IsPrivate)
	errorhandler.CheckError(err)
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
	rows, err := db.Query("SELECT name, description, is_private FROM repository WHERE user_id = ?", userID)
	errorhandler.CheckError(err)

	var grfur GetReposFromUserIDResponse
	var rds ReposDetailStruct

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description, &rds.IsPrivate)
		errorhandler.CheckError(err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return &grfur
}

// CheckRepoExists ...
func CheckRepoExists(db *sql.DB, reponame string) bool {
	rows, err := db.Query("SELECT id FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError(err)

	var repoID int

	if rows.Next() {
		err = rows.Scan(&repoID)
		errorhandler.CheckError(err)
	}

	if repoID != 0 {
		return true
	}

	return false
}

// RepoTypeStruct struct
type RepoTypeStruct struct {
	Reponame string
}

// GetRepoType ...
func GetRepoType(db *sql.DB, rts *RepoTypeStruct) bool {
	rows, err := db.Query("SELECT is_private FROM repository WHERE name = ?", rts.Reponame)
	errorhandler.CheckError(err)

	var isPrivate bool

	if rows.Next() {
		err = rows.Scan(&isPrivate)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return isPrivate
}

// CheckRepoAccessFromUserID ...
func CheckRepoAccessFromUserID(db *sql.DB, userID int) bool {
	rows, err := db.Query("SELECT name FROM repository WHERE user_id = ?", userID)
	errorhandler.CheckError(err)

	var reponame string

	if rows.Next() {
		err = rows.Scan(&reponame)
		errorhandler.CheckError(err)
	}
	rows.Close()

	if reponame != "" {
		return true
	}

	return false
}

// GetUserIDFromReponame ...
func GetUserIDFromReponame(db *sql.DB, reponame string) int {
	rows, err := db.Query("SELECT user_id FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError(err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return userID
}
