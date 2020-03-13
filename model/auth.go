package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateAccount ...
func CreateAccount(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS account (id INTEGER PRIMARY KEY, username TEXT UNIQUE NOT NULL, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, jwt_token TEXT NOT NULL, is_admin BOOLEAN DEFAULT 0)")
	errorhandler.CheckError("Error on model create account", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create account exec", err)
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
	errorhandler.CheckError("Error on model insert account", err)

	_, err = stmt.Exec(cas.Username, cas.Email, cas.PasswordHash, cas.Token, cas.IsAdmin)
	errorhandler.CheckError("Error on model insert account exec", err)
}

// GetUserIDFromToken ...
func GetUserIDFromToken(db *sql.DB, token string) int {
	rows, err := db.Query("SELECT id FROM account WHERE jwt_token = ?", token)
	errorhandler.CheckError("Error on model get userid from token", err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError("Error on model get userid from token rows scan", err)
	}
	rows.Close()

	return userID
}

// GetUsernameFromToken ...
func GetUsernameFromToken(db *sql.DB, token string) string {
	rows, err := db.Query("SELECT username FROM account WHERE jwt_token = ?", token)
	errorhandler.CheckError("Error on model get username from token", err)

	var username string

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError("Error on model get username from token row scan", err)
	}
	rows.Close()

	return username
}

// GetUsernameFromUserID ...
func GetUsernameFromUserID(db *sql.DB, userID int) string {
	rows, err := db.Query("SELECT username FROM account WHERE id = ?", userID)
	errorhandler.CheckError("Error on model get username from userid", err)

	var username string

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError("Error on model get username from userid rows scan", err)
	}
	rows.Close()

	return username
}

// GetEmailFromUsername ...
func GetEmailFromUsername(db *sql.DB, username string) string {
	rows, err := db.Query("SELECT email FROM account WHERE username = ?", username)
	errorhandler.CheckError("Error on model get email from username", err)

	var email string

	if rows.Next() {
		err = rows.Scan(&email)
		errorhandler.CheckError("Error on model get email from username rows scan", err)
	}
	rows.Close()

	return email
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
	errorhandler.CheckError("Error on model select password hash and jwt token", err)

	var sphjwtr SelectPasswordHashAndJWTTokenResponse

	if rows.Next() {
		err = rows.Scan(&sphjwtr.PasswordHash, &sphjwtr.Token)
		errorhandler.CheckError("Error on model select password hash and jwt token rows scan", err)
	}
	rows.Close()

	return &sphjwtr
}

// CheckIfFirstUserExists ...
func CheckIfFirstUserExists(db *sql.DB) bool {
	rows, err := db.Query("SELECT username from account WHERE id = ?", 1)
	errorhandler.CheckError("Error on model check if first user exists", err)

	var username string
	userExists := false

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError("Error on model check if first user exists rows scan", err)
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
	errorhandler.CheckError("Error on model reset user password by username", err)

	_, err = stmt.Exec(resetPass.PasswordHash, resetPass.JwtToken, resetPass.Username)
	errorhandler.CheckError("Error on model reset user password by username exec", err)
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
	errorhandler.CheckError("Error on model reset user password by email", err)

	_, err = stmt.Exec(resetPass.PasswordHash, resetPass.JwtToken, resetPass.Email)
	errorhandler.CheckError("Error on model reset user password by email exec", err)
}

// DeleteUserbyUsername ...
func DeleteUserbyUsername(db *sql.DB, username string) {
	stmt, err := db.Prepare("DELETE FROM account WHERE username = ?")
	errorhandler.CheckError("Error on model delete user by username", err)

	_, err = stmt.Exec(username)
	errorhandler.CheckError("Error on model delete user by username exec", err)
}

// DeleteUserbyEmail ...
func DeleteUserbyEmail(db *sql.DB, email string) {
	stmt, err := db.Prepare("DELETE FROM account WHERE email = ?")
	errorhandler.CheckError("Error on model delete user by email", err)

	_, err = stmt.Exec(email)
	errorhandler.CheckError("Error on model delete user by email exec", err)
}

// CreateSSHPubKey ...
func CreateSSHPubKey(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS ssh (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, title TEXT NOT NULL, authorized_key TEXT UNIQUE NOT NULL, fingerprint TEXT UNIQUE NOT NULL, FOREIGN KEY (user_id) REFERENCES account (id) ON DELETE CASCADE)")
	errorhandler.CheckError("Error on model create ssh pub key", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create ssh pub key exec", err)
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
	errorhandler.CheckError("Error on model insert ssh pub key", err)

	_, err = stmt.Exec(ispk.UserID, ispk.Title, ispk.AuthKey, ispk.Fingerprint)
	errorhandler.CheckError("Error on model insert ssh pub key exec", err)
}

// SSHKeysResponse struct
type SSHKeysResponse struct {
	SSHKeys []SSHDetail
}

// SSHDetail struct
type SSHDetail struct {
	Title       string
	Fingerprint string
}

// GetSSHKeysFromUserID ...
func GetSSHKeysFromUserId(db *sql.DB, userID int) *SSHKeysResponse {
	rows, err := db.Query("SELECT title, fingerprint FROM ssh WHERE user_id = ?", userID)
	errorhandler.CheckError("Error on model get ssh key from userid", err)

	var sdr SSHDetail
	var skr SSHKeysResponse

	for rows.Next() {
		err = rows.Scan(&sdr.Title, &sdr.Fingerprint)
		errorhandler.CheckError("Error on model get ssh key from userid rows scan", err)

		skr.SSHKeys = append(skr.SSHKeys, sdr)
	}
	rows.Close()

	return &skr
}

type SSHAllAuthKeysResponse struct {
	UserIDs  []string
	AuthKeys []string
}

// GetSSHAllAuthKeys ...
func GetSSHAllAuthKeys(db *sql.DB) *SSHAllAuthKeysResponse {
	rows, err := db.Query("SELECT user_id, authorized_key FROM ssh")
	errorhandler.CheckError("Error on model get ssh all auth keys", err)

	var userID, authKey string
	var userIDs, authKeys []string

	for rows.Next() {
		err = rows.Scan(&userID, &authKey)
		errorhandler.CheckError("Error on model get ssh all auth keys rows scan", err)

		userIDs = append(userIDs, userID)
		authKeys = append(authKeys, authKey)
	}
	rows.Close()

	saks := &SSHAllAuthKeysResponse{
		userIDs,
		authKeys,
	}

	return saks
}
