package auth

import (
	"database/sql"

	cError "sorcia/error"
)

// CreateAccount ...
func CreateAccount(db *sql.DB) {
	_, err := db.Query("CREATE TABLE IF NOT EXISTS account (id BIGSERIAL PRIMARY KEY, username VARCHAR(255) UNIQUE NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password_hash varchar(255) NOT NULL, jwt_token VARCHAR(255) NOT NULL, is_admin BOOLEAN NOT NULL DEFAULT FALSE)")
	cError.CheckError(err)
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
	cError.CheckError(err)
}

// GetUserIDFromToken ...
func GetUserIDFromToken(db *sql.DB, token string) int {
	rows, err := db.Query("SELECT id FROM account WHERE jwt_token = $1", token)
	cError.CheckError(err)

	var userID int

	if rows.Next() {
		err := rows.Scan(&userID)
		cError.CheckError(err)
	}
	rows.Close()

	return userID
}
