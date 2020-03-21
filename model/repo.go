package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateRepo ...
func CreateRepo(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS repository (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, name TEXT UNIQUE NOT NULL, description TEXT, is_private BOOLEAN DEFAULT 0, FOREIGN KEY (user_id) REFERENCES account (id) ON DELETE CASCADE)")
	errorhandler.CheckError("Error on model create repo", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create repo exec", err)
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
	errorhandler.CheckError("Error on model insert repo", err)

	_, err = stmt.Exec(crs.UserID, crs.Name, crs.Description, crs.IsPrivate)
	errorhandler.CheckError("Error on model insert repo exec", err)
}

// UpdateRepoStruct struct
type UpdateRepoStruct struct {
	RepoID      int
	NewName     string
	Description string
	IsPrivate   int
}

// UpdateRepo ...
func UpdateRepo(db *sql.DB, urs UpdateRepoStruct) {
	stmt, err := db.Prepare("UPDATE repository SET name = ?, description = ?, is_private = ? WHERE id = ?")
	errorhandler.CheckError("Error on model update repo", err)

	_, err = stmt.Exec(urs.NewName, urs.Description, urs.IsPrivate, urs.RepoID)
	errorhandler.CheckError("Error on model update repo exec", err)
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
	errorhandler.CheckError("Error on model get repos from user id", err)

	var grfur GetReposFromUserIDResponse
	var rds ReposDetailStruct

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description, &rds.IsPrivate)
		errorhandler.CheckError("Error on model get repos from user id rows scan", err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	rows, err = db.Query("SELECT name, description FROM repository WHERE is_private = ?", false)
	errorhandler.CheckError("Error on model get repos from userID --public-- repos", err)

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description)
		errorhandler.CheckError("Error on model get repos from userID --public-- repos rows scan", err)
		rds.IsPrivate = "false"
		if len(grfur.Repositories) == 0 {
			grfur.Repositories = append(grfur.Repositories, rds)
		} else {
			for _, repo := range grfur.Repositories {
				if repo.Name != rds.Name {
					grfur.Repositories = append(grfur.Repositories, rds)
				}
			}
		}
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
	errorhandler.CheckError("Error on model get all public repos", err)

	var grfur GetAllPublicReposResponse
	var rds ReposDetail

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description)
		errorhandler.CheckError("Error on model get all public repos rows scan", err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return &grfur
}

// GetRepoDescriptionFromRepoName ...
func GetRepoDescriptionFromRepoName(db *sql.DB, reponame string) string {
	rows, err := db.Query("SELECT description FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError("Error on model get repo description from reponame", err)

	var repoDescription string
	if rows.Next() {
		err = rows.Scan(&repoDescription)
		errorhandler.CheckError("Error on model get repo description from reponame rows scan", err)
	}
	rows.Close()

	return repoDescription
}

// GetRepoIDFromReponame ...
func GetRepoIDFromReponame(db *sql.DB, reponame string) int {
	rows, err := db.Query("SELECT id FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError("Error on model get repo id from reponame", err)

	var repoID int
	if rows.Next() {
		err = rows.Scan(&repoID)
		errorhandler.CheckError("Error on model get repo id from reponame rows scan", err)
	}
	rows.Close()

	return repoID
}

// CheckRepoExists ...
func CheckRepoExists(db *sql.DB, reponame string) bool {
	rows, err := db.Query("SELECT id FROM repository WHERE name = ?", reponame)
	errorhandler.CheckError("Error on model check repo exists", err)

	var repoID int

	if rows.Next() {
		err = rows.Scan(&repoID)
		errorhandler.CheckError("Error on model check repo exists rows scan", err)
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
	errorhandler.CheckError("Error on model get repo type", err)

	var isPrivate bool

	if rows.Next() {
		err = rows.Scan(&isPrivate)
		errorhandler.CheckError("Error on model get repo type rows scan", err)
	}
	rows.Close()

	return isPrivate
}

// CheckRepoAccessFromUserID ...
func CheckRepoAccessFromUserIDAndReponame(db *sql.DB, userID int, reponame string) bool {
	rows, err := db.Query("SELECT id FROM repository WHERE user_id = ? AND name = ?", userID, reponame)
	errorhandler.CheckError("Error on model check repo access from userid and reponame", err)

	var id int

	if rows.Next() {
		err = rows.Scan(&id)
		errorhandler.CheckError("Error on model check repo access from userid and reponame rows scan", err)
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
	errorhandler.CheckError("Error on model get userid from reponame", err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError("Error on model get userid from reponame rows scan", err)
	}
	rows.Close()

	return userID
}
