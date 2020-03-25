package model

import (
	"database/sql"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/setting"
)

// CreateAccount ...
func CreateAccount(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS account (id INTEGER PRIMARY KEY, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, jwt_token TEXT NOT NULL, can_create_repo BOOLEAN DEFAULT 0, is_admin BOOLEAN DEFAULT 0)")
	errorhandler.CheckError("Error on model create account", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create account exec", err)
}

// CreateAccountStruct struct
type CreateAccountStruct struct {
	Username      string
	PasswordHash  string
	Token         string
	CanCreateRepo int
	IsAdmin       int
}

// InsertAccount ...
func InsertAccount(db *sql.DB, cas CreateAccountStruct) {
	stmt, err := db.Prepare("INSERT INTO account (username, password_hash, jwt_token, can_create_repo, is_admin) VALUES (?, ?, ?, ?, ?)")
	errorhandler.CheckError("Error on model insert account", err)

	_, err = stmt.Exec(cas.Username, cas.PasswordHash, cas.Token, cas.CanCreateRepo, cas.IsAdmin)
	errorhandler.CheckError("Error on model insert account exec", err)
}

// RevokeCanCreateRepo ...
func RevokeCanCreateRepo(db *sql.DB, username string) {
	stmt, err := db.Prepare("UPDATE account SET can_create_repo = ? WHERE username = ?")
	errorhandler.CheckError("Error on model revoke can create repo", err)

	_, err = stmt.Exec(false, username)
	errorhandler.CheckError("Error on model revoke can create repo exec", err)
}

// AddCanCreateRepo ...
func AddCanCreateRepo(db *sql.DB, username string) {
	stmt, err := db.Prepare("UPDATE account SET can_create_repo = ? WHERE username = ?")
	errorhandler.CheckError("Error on model revoke can create repo", err)

	_, err = stmt.Exec(true, username)
	errorhandler.CheckError("Error on model revoke can create repo exec", err)
}

type Users struct {
	Users []User
}

type User struct {
	Username      string
	CanCreateRepo bool
	IsAdmin       bool
}

// GetAllUsers
func GetAllUsers(db *sql.DB) Users {
	rows, err := db.Query("SELECT username, can_create_repo, is_admin FROM account")
	errorhandler.CheckError("Error on model get all users", err)

	var user User
	var users Users

	for rows.Next() {
		err = rows.Scan(&user.Username, &user.CanCreateRepo, &user.IsAdmin)
		errorhandler.CheckError("Error on model get all users rows scan", err)

		users.Users = append(users.Users, user)
	}
	rows.Close()

	return users
}

// CheckifUserCanCreateRepo ...
func CheckifUserCanCreateRepo(db *sql.DB, userID int) bool {
	rows, err := db.Query("SELECT can_create_repo FROM account WHERE id = ?", userID)
	errorhandler.CheckError("Error on model check if user can create repo", err)

	var canCreateRepo bool

	if rows.Next() {
		err = rows.Scan(&canCreateRepo)
		errorhandler.CheckError("Error on model check if user can create repo rows scan", err)
	}
	rows.Close()

	return canCreateRepo
}

// CheckifUserCanCreateRepo ...
func CheckifUserIsAnAdmin(db *sql.DB, userID int) bool {
	rows, err := db.Query("SELECT is_admin FROM account WHERE id = ?", userID)
	errorhandler.CheckError("Error on model check if user is an admin", err)

	var isAdmin bool

	if rows.Next() {
		err = rows.Scan(&isAdmin)
		errorhandler.CheckError("Error on model check if user is an admin rows scan", err)
	}
	rows.Close()

	return isAdmin
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

// GetUserIDFromUsername ...
func GetUserIDFromUsername(db *sql.DB, username string) int {
	rows, err := db.Query("SELECT id FROM account WHERE username = ?", username)
	errorhandler.CheckError("Error on model get userid from username", err)

	var userID int

	if rows.Next() {
		err = rows.Scan(&userID)
		errorhandler.CheckError("Error on model get userid from username rows scan", err)
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

// ResetUsernameByUserID ...
func ResetUsernameByUserID(db *sql.DB, newUsername string, userID int) {
	stmt, err := db.Prepare("UPDATE account SET username = ? WHERE id = ?")
	errorhandler.CheckError("Error on model reset username by userID", err)

	_, err = stmt.Exec(newUsername, userID)
	errorhandler.CheckError("Error on model reset username by userID exec", err)
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

// DeleteUserbyUsername ...
func DeleteUserbyUsername(db *sql.DB, username string) {
	stmt, err := db.Prepare("DELETE FROM account WHERE username = ?")
	errorhandler.CheckError("Error on model delete user by username", err)

	_, err = stmt.Exec(username)
	errorhandler.CheckError("Error on model delete user by username exec", err)
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

// DeleteMetaKeyByID ...
func DeleteMetaKeyByID(db *sql.DB, id int) {
	stmt, err := db.Prepare("DELETE FROM ssh WHERE id = ?")
	errorhandler.CheckError("Error on model delete meta key by id", err)

	_, err = stmt.Exec(id)
	errorhandler.CheckError("Error on model delete meta key by id exec", err)
}

// SSHKeysResponse struct
type SSHKeysResponse struct {
	SSHKeys []SSHDetail
}

// SSHDetail struct
type SSHDetail struct {
	ID          int
	Title       string
	Fingerprint string
}

// GetSSHKeysFromUserID ...
func GetSSHKeysFromUserId(db *sql.DB, userID int) *SSHKeysResponse {
	rows, err := db.Query("SELECT id, title, fingerprint FROM ssh WHERE user_id = ?", userID)
	errorhandler.CheckError("Error on model get ssh key from userid", err)

	var sdr SSHDetail
	var skr SSHKeysResponse

	for rows.Next() {
		err = rows.Scan(&sdr.ID, &sdr.Title, &sdr.Fingerprint)
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

// CreateSiteSettings ...
func CreateSiteSettings(db *sql.DB) {
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS site_settings (id INTEGER PRIMARY KEY, title TEXT NOT NULL, favicon TEXT, logo TEXT, logo_width TEXT, logo_height TEXT, style TEXT DEFAULT 'default')")
	errorhandler.CheckError("Error on model create site", err)

	_, err = stmt.Exec()
	errorhandler.CheckError("Error on model create site exec", err)
}

// CreateSiteSettingsStruct struct
type CreateSiteSettingsStruct struct {
	Title      string
	Favicon    string
	Logo       string
	LogoWidth  string
	LogoHeight string
	Style      string
}

// InsertSiteSettings ...
func InsertSiteSettings(db *sql.DB, css CreateSiteSettingsStruct) {
	stmt, err := db.Prepare("INSERT INTO site_settings (title, favicon, logo, logo_width, logo_height, style) VALUES (?, ?, ?, ?, ?, ?)")
	errorhandler.CheckError("Error on model insert site", err)

	_, err = stmt.Exec(css.Title, css.Favicon, css.Logo, css.LogoWidth, css.LogoHeight, css.Style)
	errorhandler.CheckError("Error on model insert site exec", err)
}

// CheckIFSiteSettingsExists ...
func CheckIFSiteSettingsExists(db *sql.DB) bool {
	rows, err := db.Query("SELECT title FROM site_settings WHERE id = ?", 1)
	errorhandler.CheckError("Error on model check if site settings exists", err)

	var title string
	isExists := false

	if rows.Next() {
		err = rows.Scan(&title)
		errorhandler.CheckError("Error on model check if site_settings settings exists rows scan", err)
	}
	rows.Close()

	if title != "" {
		isExists = true
	}

	return isExists
}

type GetSiteSettingsResponse struct {
	Title      string
	Favicon    string
	Logo       string
	LogoWidth  string
	LogoHeight string
	Style      string
}

// GetSiteSettings ...
func GetSiteSettings(db *sql.DB, conf *setting.BaseStruct) *GetSiteSettingsResponse {
	rows, err := db.Query("SELECT title, favicon, logo, logo_width, logo_height, style FROM site_settings WHERE id = ?", 1)
	errorhandler.CheckError("Error on model get site settings", err)

	var title, favicon, logo, logoWidth, logoHeight, style string
	gssr := GetSiteSettingsResponse{}

	if rows.Next() {
		err = rows.Scan(&title, &favicon, &logo, &logoWidth, &logoHeight, &style)
		errorhandler.CheckError("Error on model get site_settings settings rows scan", err)

		faviconSplit := strings.Split(favicon, conf.Paths.UploadAssetPath)
		if len(faviconSplit) > 1 {
			favicon = faviconSplit[1]
		}

		logoSplit := strings.Split(logo, conf.Paths.UploadAssetPath)
		if len(logoSplit) > 1 {
			logo = logoSplit[1]
		}

		gssr = GetSiteSettingsResponse{
			Title:      title,
			Favicon:    favicon,
			Logo:       logo,
			LogoWidth:  logoWidth,
			LogoHeight: logoHeight,
			Style:      style,
		}
	}
	rows.Close()

	return &gssr
}

// GetSiteStyle ...
func GetSiteStyle(db *sql.DB) string {
	rows, err := db.Query("SELECT style FROM site_settings WHERE id = ?", 1)
	errorhandler.CheckError("Error on model get site style", err)

	style := "default"
	if rows.Next() {
		err = rows.Scan(&style)
		errorhandler.CheckError("Error on model get site style rows scan", err)
	}
	rows.Close()

	return style
}

// GetSiteFavicon
func GetSiteFavicon(db *sql.DB) string {
	rows, err := db.Query("SELECT favicon FROM site_settings WHERE id = ?", 1)
	errorhandler.CheckError("Error on model get site favicon", err)

	var favicon string
	if rows.Next() {
		err = rows.Scan(&favicon)
		errorhandler.CheckError("Error on model get site favicon rows scan", err)
	}
	rows.Close()

	return favicon
}

// GetSiteLogo
func GetSiteLogo(db *sql.DB) string {
	rows, err := db.Query("SELECT logo FROM site_settings WHERE id = ?", 1)
	errorhandler.CheckError("Error on model get site logo", err)

	var logo string
	if rows.Next() {
		err = rows.Scan(&logo)
		errorhandler.CheckError("Error on model get site logo rows scan", err)
	}
	rows.Close()

	return logo
}

// UpdateSiteTitle ...
func UpdateSiteTitle(db *sql.DB, title string) {
	stmt, err := db.Prepare("UPDATE site_settings SET title = ? WHERE id = 1")
	errorhandler.CheckError("Error on model update site title", err)

	_, err = stmt.Exec(title)
	errorhandler.CheckError("Error on model update site title exec", err)
}

// UpdateSiteFavicon ...
func UpdateSiteFavicon(db *sql.DB, favicon string) {
	stmt, err := db.Prepare("UPDATE site_settings SET favicon = ? WHERE id = 1")
	errorhandler.CheckError("Error on model update site favicon", err)

	_, err = stmt.Exec(favicon)
	errorhandler.CheckError("Error on model update site favicon exec", err)
}

// UpdateSiteLogo ...
func UpdateSiteLogo(db *sql.DB, logo, logoWidth, logoHeight string) {
	stmt, err := db.Prepare("UPDATE site_settings SET logo = ? WHERE id = 1")
	errorhandler.CheckError("Error on model update site logo", err)

	_, err = stmt.Exec(logo)
	errorhandler.CheckError("Error on model update site logo exec", err)
}

// UpdateSiteStyle ...
func UpdateSiteStyle(db *sql.DB, style string) {
	stmt, err := db.Prepare("UPDATE site_settings SET style = ? WHERE id = 1")
	errorhandler.CheckError("Error on model update site style", err)

	_, err = stmt.Exec(style)
	errorhandler.CheckError("Error on model update site style exec", err)
}
