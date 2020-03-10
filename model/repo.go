package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateRepo ...
func CreateRepo(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS repository (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, name TEXT UNIQUE NOT NULL, description TEXT, is_private BOOLEAN DEFAULT 0, FOREIGN KEY (user_id) REFERENCES account (id) ON DELETE CASCADE)")
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

type GetAllPublicReposResponse struct {
	Repositories []ReposDetail
}

type ReposDetail struct {
	Name        string
	Description string
}

// GetAllPublicRepos ...
func GetAllPublicRepos(db *sql.DB) *GetAllPublicReposResponse {
	rows, err := db.Query("SELECT name, description FROM repository WHERE is_private = ?", false)
	errorhandler.CheckError(err)

	var grfur GetAllPublicReposResponse
	var rds ReposDetail

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description)
		errorhandler.CheckError(err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return &grfur
}

// GetRepoDescriptionFromRepoName ...
func GetRepoDescriptionFromRepoName(db *sql.DB, reponame string) string {
	rows, err := db.Query("SELECT description FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError(err)

	var repoDescription string
	if rows.Next() {
		err = rows.Scan(&repoDescription)
	}
	rows.Close()

	return repoDescription
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
	rows.Close()

	if repoID != 0 {
		return true
	}

	return false
}

// GetRepoType ...
func GetRepoType(db *sql.DB, reponame string) bool {
	rows, err := db.Query("SELECT is_private FROM repository WHERE name = ?", reponame)
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
func CheckRepoAccessFromUserIDAndReponame(db *sql.DB, userID int, reponame string) bool {
	rows, err := db.Query("SELECT id FROM repository WHERE user_id = ? AND name = ?", userID, reponame)
	errorhandler.CheckError(err)

	var id int

	if rows.Next() {
		err = rows.Scan(&id)
		errorhandler.CheckError(err)
	}
	rows.Close()

	if id > 0 {
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
