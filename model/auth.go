package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateAccount ...
func CreateAccount(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS account (id INTEGER PRIMARY KEY, username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, jwt_token TEXT NOT NULL, is_admin BOOLEAN DEFAULT 0)")
	errorhandler.CheckError(err)

	_, err = stmt.Exec()
	errorhandler.CheckError(err)
}

// CreateAccountStruct struct
type CreateAccountStruct struct {
	Username     string
	Email        string
	PasswordHash string
	Token        string
	IsAdmin      int
}

// InsertAccount ...
func InsertAccount(db *sql.DB, cas CreateAccountStruct) {
	stmt, err := db.Prepare("INSERT INTO account (username, email, password_hash, jwt_token, is_admin) VALUES (?, ?, ?, ?, ?)")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(cas.Username, cas.Email, cas.PasswordHash, cas.Token, cas.IsAdmin)
	errorhandler.CheckError(err)
}

// GetUserIDFromToken ...
func GetUserIDFromToken(db *sql.DB, token string) int {
	rows, err := db.Query("SELECT id FROM account WHERE jwt_token = ?", token)
	errorhandler.CheckError(err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return userID
}

// GetUsernameFromToken ...
func GetUsernameFromToken(db *sql.DB, token string) string {
	rows, err := db.Query("SELECT username FROM account WHERE jwt_token = ?", token)
	errorhandler.CheckError(err)

	var username string

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return username
}

// GetUsernameFromUserID ...
func GetUsernameFromUserID(db *sql.DB, userID int) string {
	rows, err := db.Query("SELECT username FROM account WHERE id = ?", userID)
	errorhandler.CheckError(err)

	var username string

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return username
}

// SelectPasswordHashAndJWTTokenStruct struct
type SelectPasswordHashAndJWTTokenStruct struct {
	Username string
}

// SelectPasswordHashAndJWTTokenResponse struct
type SelectPasswordHashAndJWTTokenResponse struct {
	PasswordHash string
	Token        string
}

// SelectPasswordHashAndJWTToken ...
func SelectPasswordHashAndJWTToken(db *sql.DB, sphjwt SelectPasswordHashAndJWTTokenStruct) *SelectPasswordHashAndJWTTokenResponse {
	// Search for username in the 'account' table with the given string
	rows, err := db.Query("SELECT password_hash, jwt_token FROM account WHERE username = ?", sphjwt.Username)
	errorhandler.CheckError(err)

	var sphjwtr SelectPasswordHashAndJWTTokenResponse

	if rows.Next() {
		err = rows.Scan(&sphjwtr.PasswordHash, &sphjwtr.Token)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return &sphjwtr
}

// CheckIfFirstUserExists ...
func CheckIfFirstUserExists(db *sql.DB) bool {
	rows, err := db.Query("SELECT username from account WHERE id = ?", 1)
	errorhandler.CheckError(err)

	var username string
	userExists := false

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError(err)
	}
	rows.Close()

	if username != "" {
		userExists = true
	}

	return userExists
}

// ResetUserPasswordbyUsernameStruct struct
type ResetUserPasswordbyUsernameStruct struct {
	PasswordHash string
	JwtToken     string
	Username     string
}

// ResetUserPasswordbyUsername ...
func ResetUserPasswordbyUsername(db *sql.DB, resetPass ResetUserPasswordbyUsernameStruct) {
	stmt, err := db.Prepare("UPDATE account SET password_hash = ?, jwt_token = ? WHERE username = ?")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(resetPass.PasswordHash, resetPass.JwtToken, resetPass.Username)
	errorhandler.CheckError(err)
}

// ResetUserPasswordbyEmailStruct struct
type ResetUserPasswordbyEmailStruct struct {
	PasswordHash string
	JwtToken     string
	Email        string
}

// ResetUserPasswordbyEmail ...
func ResetUserPasswordbyEmail(db *sql.DB, resetPass ResetUserPasswordbyEmailStruct) {
	stmt, err := db.Prepare("UPDATE account SET password_hash = ?, jwt_token = ? WHERE email = ?")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(resetPass.PasswordHash, resetPass.JwtToken, resetPass.Email)
	errorhandler.CheckError(err)
}

// DeleteUserbyUsername ...
func DeleteUserbyUsername(db *sql.DB, username string) {
	stmt, err := db.Prepare("DELETE FROM account WHERE username = ?")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(username)
	errorhandler.CheckError(err)
}

// DeleteUserbyEmail ...
func DeleteUserbyEmail(db *sql.DB, email string) {
	stmt, err := db.Prepare("DELETE FROM account WHERE email = ?")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(email)
	errorhandler.CheckError(err)
}

// CreateSSHPubKey ...
func CreateSSHPubKey(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS ssh (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, title TEXT UNIQUE NOT NULL, authorized_key TEXT UNIQUE NOT NULL, fingerprint TEXT UNIQUE NOT NULL, FOREIGN KEY (user_id) REFERENCES account (id) ON DELETE CASCADE)")
	errorhandler.CheckError(err)

	_, err = stmt.Exec()
	errorhandler.CheckError(err)
}

// InsertSSHPubKey struct
type InsertSSHPubKeyStruct struct {
	AuthKey     string
	Title       string
	Fingerprint string
	UserID      int
}

// InsertRepo ...
func InsertSSHPubKey(db *sql.DB, ispk InsertSSHPubKeyStruct) {
	stmt, err := db.Prepare("INSERT INTO ssh (user_id, title, authorized_key, fingerprint) VALUES (?, ?, ?, ?)")
	errorhandler.CheckError(err)

	_, err = stmt.Exec(ispk.UserID, ispk.Title, ispk.AuthKey, ispk.Fingerprint)
	errorhandler.CheckError(err)
}

// SSHKeysResponse struct
type SSHKeysResponse struct {
	SSHKeys []SSHDetail
}

// SSHDetailResponse struct
type SSHDetail struct {
	Title       string
	Fingerprint string
}

// GetSSHKeys ...
func GetSSHKeys(db *sql.DB, userID int) *SSHKeysResponse {
	rows, err := db.Query("SELECT title, fingerprint FROM ssh WHERE user_id = ?", userID)
	errorhandler.CheckError(err)

	var sdr SSHDetail
	var skr SSHKeysResponse

	for rows.Next() {
		err = rows.Scan(&sdr.Title, &sdr.Fingerprint)
		errorhandler.CheckError(err)

		skr.SSHKeys = append(skr.SSHKeys, sdr)
	}
	rows.Close()

	return &skr
}
