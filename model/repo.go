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

// DeleteRepobyReponame ...
func DeleteRepobyReponame(db *sql.DB, reponame string) {
	stmt, err := db.Prepare("DELETE FROM repository WHERE name = ?")
	errorhandler.CheckError("Error on model delete repository by reponame", err)

	_, err = stmt.Exec(reponame)
	errorhandler.CheckError("Error on model delete repository by reponame exec", err)
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

// CreateRepoMembers ...
func CreateRepoMembers(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS repository_members (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, repo_id INTEGER NOT NULL, permission TEXT NOT NULL, FOREIGN KEY (repo_id) REFERENCES repository (id) ON DELETE CASCADE)")
	errorhandler.CheckError("Error on model create repo members", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create repo members exec", err)
}

// CreateRepoMember struct
type CreateRepoMember struct {
	UserID     int
	RepoID     int
	Permission string
}

// InsertRepoMember ...
func InsertRepoMember(db *sql.DB, crm CreateRepoMember) {
	stmt, err := db.Prepare("INSERT INTO repository_members (user_id, repo_id, permission) VALUES (?, ?, ?)")
	errorhandler.CheckError("Error on model insert repo member", err)

	_, err = stmt.Exec(crm.UserID, crm.RepoID, crm.Permission)
	errorhandler.CheckError("Error on model insert repo member exec", err)
}

// GetRepoMembersStruct struct
type GetRepoMembersStruct struct {
	RepoMembers []RepoMember
}

// RepoMember struct
type RepoMember struct {
	UserID     int
	Username   string
	Permission string
	IsOwner    bool
}

// GetRepoMembers ...
func GetRepoMembers(db *sql.DB, repoID int) GetRepoMembersStruct {
	rows, err := db.Query("SELECT user_id, permission FROM repository_members WHERE repo_id = ?", repoID)
	errorhandler.CheckError("Error on model get repos members", err)

	var rm RepoMember
	var grms GetRepoMembersStruct

	for rows.Next() {
		err := rows.Scan(&rm.UserID, &rm.Permission)
		errorhandler.CheckError("Error on model get repo members rows scan", err)

		rm.Username = GetUsernameFromUserID(db, rm.UserID)
		rm.IsOwner = false

		grms.RepoMembers = append(grms.RepoMembers, rm)
	}
	rows.Close()

	rows, err = db.Query("SELECT user_id FROM repository WHERE id = ?", repoID)
	errorhandler.CheckError("Error on model get repos members - repository", err)

	if rows.Next() {
		err := rows.Scan(&rm.UserID)
		errorhandler.CheckError("Error on model get repo members rows scan", err)

		rm.Username = GetUsernameFromUserID(db, rm.UserID)
		rm.Permission = "read/write"
		rm.IsOwner = true

		grms.RepoMembers = append(grms.RepoMembers, rm)
	}
	rows.Close()

	return grms
}

// GetRepoIDsOnRepoMembersUsingUserID ...
func GetRepoIDsOnRepoMembersUsingUserID(db *sql.DB, userID int) []int {
	rows, err := db.Query("SELECT repo_id FROM repository_members WHERE user_id = ?", userID)
	errorhandler.CheckError("Error on model get repoids on repomembers using user id", err)

	var repoIDs []int
	var repoID int
	for rows.Next() {
		err := rows.Scan(&repoID)
		errorhandler.CheckError("Error on model get repoids on repomembers using user id rows scan", err)

		repoIDs = append(repoIDs, repoID)
	}

	return repoIDs
}

// GetRepoMemberIDFromUserID ...
func GetRepoMemberIDFromUserID(db *sql.DB, userID int) []int {
	rows, err := db.Query("SELECT id FROM repository_members WHERE user_id = ?", userID)
	errorhandler.CheckError("Error on model get repos members id from user id", err)

	var repoMembersID []int
	var repoMemberID int

	for rows.Next() {
		err := rows.Scan(&repoMemberID)
		errorhandler.CheckError("Error on model get repo members id from user id rows scan", err)

		repoMembersID = append(repoMembersID, repoMemberID)
	}
	rows.Close()

	return repoMembersID
}

// DeleteRepoMemberByID ...
func DeleteRepoMemberByID(db *sql.DB, id int) {
	stmt, err := db.Prepare("DELETE FROM repository_members WHERE id = ?")
	errorhandler.CheckError("Error on model delete repository_member by id", err)

	_, err = stmt.Exec(id)
	errorhandler.CheckError("Error on model delete repository_member by id exec", err)
}

// GetReposStruct struct
type GetReposStruct struct {
	Repositories []RepoDetailStruct
}

// ReposDetailStruct struct
type RepoDetailStruct struct {
	Name        string
	Description string
	IsPrivate   string
}

// GetReposFromUserID ...
func GetReposFromUserID(db *sql.DB, userID int) GetReposStruct {
	rows, err := db.Query("SELECT name, description, is_private FROM repository WHERE user_id = ?", userID)
	errorhandler.CheckError("Error on model get repos from user id", err)

	var grfur GetReposStruct
	var rds RepoDetailStruct

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description, &rds.IsPrivate)
		errorhandler.CheckError("Error on model get repos from user id rows scan", err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return grfur
}

// GetReposFromRepoID ...
func GetRepoFromRepoID(db *sql.DB, repoID int) RepoDetailStruct {
	rows, err := db.Query("SELECT name, description, is_private FROM repository WHERE id = ?", repoID)
	errorhandler.CheckError("Error on model get repos from user id", err)

	var rds RepoDetailStruct

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description, &rds.IsPrivate)
		errorhandler.CheckError("Error on model get repos from user id rows scan", err)
	}
	rows.Close()

	return rds
}

// GetAllPublicRepos ...
func GetAllPublicRepos(db *sql.DB) GetReposStruct {
	rows, err := db.Query("SELECT name, description FROM repository WHERE is_private = ?", false)
	errorhandler.CheckError("Error on model get all public repos", err)

	var grfur GetReposStruct
	var rds RepoDetailStruct

	for rows.Next() {
		err = rows.Scan(&rds.Name, &rds.Description)
		errorhandler.CheckError("Error on model get all public repos rows scan", err)

		grfur.Repositories = append(grfur.Repositories, rds)
	}
	rows.Close()

	return grfur
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

// CheckRepoOwnerFromUserID ...
func CheckRepoOwnerFromUserIDAndReponame(db *sql.DB, userID int, reponame string) bool {
	rows, err := db.Query("SELECT id FROM repository WHERE user_id = ? AND name = ?", userID, reponame)
	errorhandler.CheckError("Error on model check repo owner from userid and reponame", err)

	var id int

	if rows.Next() {
		err = rows.Scan(&id)
		errorhandler.CheckError("Error on model check repo owner from userid and reponame rows scan", err)
	}
	rows.Close()

	if id > 0 {
		return true
	}

	return false
}

// CheckRepoMemberExistFromUserIDAndRepoID ...
func CheckRepoMemberExistFromUserIDAndRepoID(db *sql.DB, userID, repoID int) bool {
	rows, err := db.Query("SELECT id FROM repository_members WHERE user_id = ? AND repo_id = ?", userID, repoID)
	errorhandler.CheckError("Error on model check repo member exist from userid and repoid", err)

	var id int

	if rows.Next() {
		err = rows.Scan(&id)
		errorhandler.CheckError("Error on model check repo member exist from userid and repoid rows scan", err)
	}
	rows.Close()

	if id > 0 {
		return true
	}

	return false
}

// GetRepoMemberPermissionFromUserIDAndRepoID ...
func GetRepoMemberPermissionFromUserIDAndRepoID(db *sql.DB, userID, repoID int) string {
	rows, err := db.Query("SELECT permission FROM repository_members WHERE user_id = ? AND repo_id = ?", userID, repoID)
	errorhandler.CheckError("Error on model check repo permissions from userid and repoid", err)

	var permission string

	if rows.Next() {
		err = rows.Scan(&permission)
		errorhandler.CheckError("Error on model check repo permissions from userid and repoid rows scan", err)
	}
	rows.Close()

	return permission
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
