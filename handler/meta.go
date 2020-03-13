package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"
	"sorcia/util"

	"github.com/gorilla/schema"
)

type MetaResponse struct {
	IsLoggedIn       bool
	HeaderActiveMenu string
	SorciaVersion    string
	Username         string
	Email            string
}

// GetMeta ...
func GetMeta(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		username := model.GetUsernameFromToken(db, token)
		email := model.GetEmailFromUsername(db, username)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := MetaResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
			Username:         username,
			Email:            email,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// MetaKeysResponse struct
type MetaKeysResponse struct {
	IsLoggedIn       bool
	HeaderActiveMenu string
	SorciaVersion    string
	SSHKeys          *model.SSHKeysResponse
}

// GetMetaKeys ...
func GetMetaKeys(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		sshKeys := model.GetSSHKeysFromUserId(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta-keys.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := MetaKeysResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
			SSHKeys:          sshKeys,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// CreateAuthKeyRequest struct
type CreateAuthKeyRequest struct {
	Title   string `schema:"sshtitle"`
	AuthKey string `schema:"sshkey"`
}

// PostAuthKey ...
func PostAuthKey(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct, decoder *schema.Decoder) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &errorhandler.Response{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		errorhandler.CheckError("Error on json marshal", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	var createAuthKeyRequest = &CreateAuthKeyRequest{}
	err := decoder.Decode(createAuthKeyRequest, r.PostForm)
	errorhandler.CheckError("Error on auth key decode", err)

	token := w.Header().Get("sorcia-cookie-token")
	userID := model.GetUserIDFromToken(db, token)

	authKey := strings.TrimSpace(createAuthKeyRequest.AuthKey)
	fingerPrint := util.SSHFingerPrint(authKey)

	ispk := model.InsertSSHPubKeyStruct{
		AuthKey:     authKey,
		Title:       strings.TrimSpace(createAuthKeyRequest.Title),
		Fingerprint: fingerPrint,
		UserID:      userID,
	}

	model.InsertSSHPubKey(db, ispk)

	http.Redirect(w, r, "/meta/keys", http.StatusFound)
}

// GetMetaUsers ...
func GetMetaUsers(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta-users.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := MetaResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// PostPasswordRequest struct
type PostPasswordRequest struct {
	Password string `schema:"password"`
}

func MetaPostPassword(w http.ResponseWriter, r *http.Request, db *sql.DB, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")

		// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &errorhandler.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			errorhandler.CheckError("Error on json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		postPasswordRequest := &PostPasswordRequest{}
		err := decoder.Decode(postPasswordRequest, r.PostForm)
		errorhandler.CheckError("Error on post password decoder", err)

		username := model.GetUsernameFromToken(db, token)

		// Generate password hash using bcrypt
		passwordHash, err := HashPassword(postPasswordRequest.Password)
		errorhandler.CheckError("Error on password hash", err)

		// Generate JWT token using the hash password above
		jwt_token, err := GenerateJWTToken(passwordHash)
		errorhandler.CheckError("Error on generating jwt token", err)

		resetPass := model.ResetUserPasswordbyUsernameStruct{
			PasswordHash: passwordHash,
			JwtToken:     jwt_token,
			Username:     username,
		}
		model.ResetUserPasswordbyUsername(db, resetPass)
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}
