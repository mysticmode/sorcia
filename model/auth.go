package model

import (
	"database/sql"

	errorhandler "sorcia/error"
)

// CreateAccount ...
func CreateAccount(db *sql.DB) {
	_, err := db.Query("CREATE TABLE IF NOT EXISTS account (id BIGSERIAL PRIMARY KEY, username VARCHAR(255) UNIQUE NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password_hash varchar(255) NOT NULL, jwt_token VARCHAR(255) NOT NULL, is_admin BOOLEAN NOT NULL DEFAULT FALSE)")
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
	var lastInsertID int

	err := db.QueryRow("INSERT INTO account (username, email, password_hash, jwt_token, is_admin) VALUES ($1, $2, $3, $4, $5) returning id", cas.Username, cas.Email, cas.PasswordHash, cas.Token, cas.IsAdmin).Scan(&lastInsertID)
	errorhandler.CheckError(err)
}

// GetUserIDFromToken ...
func GetUserIDFromToken(db *sql.DB, token string) int {
	rows, err := db.Query("SELECT id FROM account WHERE jwt_token = $1", token)
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
	rows, err := db.Query("SELECT username FROM account WHERE jwt_token = $1", token)
	errorhandler.CheckError(err)

	var username string

	if rows.Next() {
		err = rows.Scan(&username)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return username
}

// GetUserIDFromUsername ...
func GetUserIDFromUsername(db *sql.DB, username string) int {
	rows, err := db.Query("SELECT id FROM account WHERE username = $1", username)
	errorhandler.CheckError(err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return userID
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
	rows, err := db.Query("SELECT password_hash, jwt_token FROM account WHERE username = $1", sphjwt.Username)
	errorhandler.CheckError(err)

	var sphjwtr SelectPasswordHashAndJWTTokenResponse

	if rows.Next() {
		err = rows.Scan(&sphjwtr.PasswordHash, &sphjwtr.Token)
		errorhandler.CheckError(err)
	}
	rows.Close()

	return &sphjwtr
}
