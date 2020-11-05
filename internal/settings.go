package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"image"

	// jpeg import
	_ "image/jpeg"

	// png import
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sorcia/models"
	"sorcia/pkg"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// SettingsResponse struct
type SettingsResponse struct {
	IsLoggedIn         bool
	IsAdmin            bool
	HeaderActiveMenu   string
	SorciaVersion      string
	Username           string
	Email              string
	Users              models.Users
	RegisterErrMessage string
	SiteSettings       SiteSettings
}

// GetSettings ...
func GetSettings(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		username := models.GetUsernameFromToken(db, token)

		userID := models.GetUserIDFromToken(db, token)

		layoutPage := filepath.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := filepath.Join(conf.Paths.TemplatePath, "header.html")
		metaPage := filepath.Join(conf.Paths.TemplatePath, "settings.html")
		footerPage := filepath.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		pkg.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := SettingsResponse{
			IsLoggedIn:       true,
			IsAdmin:          models.CheckifUserIsAnAdmin(db, userID),
			HeaderActiveMenu: "meta",
			SorciaVersion:    conf.Version,
			Username:         username,
			SiteSettings:     GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// SettingsKeysResponse struct
type SettingsKeysResponse struct {
	IsLoggedIn       bool
	IsAdmin          bool
	HeaderActiveMenu string
	SorciaVersion    string
	SSHKeys          *models.SSHKeysResponse
	SiteSettings     SiteSettings
}

// GetSettingsKeys ...
func GetSettingsKeys(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := models.GetUserIDFromToken(db, token)

		sshKeys := models.GetSSHKeysFromUserID(db, userID)

		layoutPage := filepath.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := filepath.Join(conf.Paths.TemplatePath, "header.html")
		metaPage := filepath.Join(conf.Paths.TemplatePath, "settings-keys.html")
		footerPage := filepath.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		pkg.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := SettingsKeysResponse{
			IsLoggedIn:       true,
			IsAdmin:          models.CheckifUserIsAnAdmin(db, userID),
			HeaderActiveMenu: "meta",
			SorciaVersion:    conf.Version,
			SSHKeys:          sshKeys,
			SiteSettings:     GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// DeleteSettingsKey ...
func DeleteSettingsKey(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	keyID := vars["keyID"]

	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		i, err := strconv.Atoi(keyID)
		pkg.CheckError("Error on converting SSH key id(string) to int on delete settings keys", err)

		models.DeleteSettingsKeyByID(db, i)
		http.Redirect(w, r, "/settings/keys", http.StatusFound)
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
func PostAuthKey(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")

		// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &pkg.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			pkg.CheckError("Error on json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		var createAuthKeyRequest = &CreateAuthKeyRequest{}
		err := decoder.Decode(createAuthKeyRequest, r.PostForm)
		pkg.CheckError("Error on auth key decode", err)

		userID := models.GetUserIDFromToken(db, token)

		authKey := strings.TrimSpace(createAuthKeyRequest.AuthKey)
		fingerPrint := pkg.SSHFingerPrint(authKey)

		ispk := models.InsertSSHPubKeyStruct{
			AuthKey:     authKey,
			Title:       strings.TrimSpace(createAuthKeyRequest.Title),
			Fingerprint: fingerPrint,
			UserID:      userID,
		}

		models.InsertSSHPubKey(db, ispk)

		http.Redirect(w, r, "/settings/keys", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// RevokeCreateRepoAccess ...
func RevokeCreateRepoAccess(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")
	vars := mux.Vars(r)
	username := vars["username"]

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := models.GetUserIDFromToken(db, token)

		if models.CheckifUserIsAnAdmin(db, userID) {
			models.RevokeCanCreateRepo(db, username)

			http.Redirect(w, r, "/settings/users", http.StatusFound)
			return
		}
	}

	http.Redirect(w, r, "/settings/users", http.StatusFound)
}

// AddCreateRepoAccess ...
func AddCreateRepoAccess(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")
	vars := mux.Vars(r)
	username := vars["username"]

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := models.GetUserIDFromToken(db, token)

		if models.CheckifUserIsAnAdmin(db, userID) {
			models.AddCanCreateRepo(db, username)

			http.Redirect(w, r, "/settings/users", http.StatusFound)
			return
		}
	}

	http.Redirect(w, r, "/settings/users", http.StatusFound)
}

// PostUserRequest struct
type PostUserRequest struct {
	Username      string `schema:"username"`
	Password      string `schema:"password"`
	CanCreateRepo string `schema:"createrepo"`
}

// PostUser ...
func PostUser(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &pkg.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			pkg.CheckError("Error on json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		var postUserRequest = &PostUserRequest{}
		err := decoder.Decode(postUserRequest, r.PostForm)
		pkg.CheckError("Error on meta post user", err)

		// Generate password hash using bcrypt
		passwordHash, err := HashPassword(postUserRequest.Password)
		pkg.CheckError("Error on post register hash password", err)

		// Generate JWT token using the hash password above
		token, err := GenerateJWTToken(passwordHash)
		pkg.CheckError("Error on post register generate jwt token", err)

		s := postUserRequest.Username

		if len(s) > 39 || len(s) < 1 {
			layoutPage := filepath.Join(conf.Paths.TemplatePath, "layout.html")
			headerPage := filepath.Join(conf.Paths.TemplatePath, "header.html")
			metaUsersPage := filepath.Join(conf.Paths.TemplatePath, "settings-users.html")
			footerPage := filepath.Join(conf.Paths.TemplatePath, "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, metaUsersPage, footerPage)
			pkg.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsLoggedIn:         false,
				ShowLoginMenu:      false,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				IsShowSignUp:       !models.CheckIfFirstUserExists(db),
				LoginErrMessage:    "",
				RegisterErrMessage: "Username is too long (maximum is 39 characters).",
				SiteSettings:       GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !pkg.IsAlnumOrHyphen(s) {
			layoutPage := filepath.Join(conf.Paths.TemplatePath, "layout.html")
			headerPage := filepath.Join(conf.Paths.TemplatePath, "header.html")
			metaUsersPage := filepath.Join(conf.Paths.TemplatePath, "settings-users.html")
			footerPage := filepath.Join(conf.Paths.TemplatePath, "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, metaUsersPage, footerPage)
			pkg.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsLoggedIn:         false,
				ShowLoginMenu:      false,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				IsShowSignUp:       !models.CheckIfFirstUserExists(db),
				LoginErrMessage:    "",
				RegisterErrMessage: "Username may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
				SiteSettings:       GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}

		canCreateRepo := 0
		if postUserRequest.CanCreateRepo != "" {
			canCreateRepo = 1
		}

		rr := models.CreateAccountStruct{
			Username:      postUserRequest.Username,
			PasswordHash:  passwordHash,
			Token:         token,
			CanCreateRepo: canCreateRepo,
			IsAdmin:       0,
		}

		models.InsertAccount(db, rr)

		http.Redirect(w, r, "/meta/users", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// GetSettingsUsers ...
func GetSettingsUsers(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := models.GetUserIDFromToken(db, token)

		users := models.GetAllUsers(db)

		layoutPage := filepath.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := filepath.Join(conf.Paths.TemplatePath, "header.html")
		metaPage := filepath.Join(conf.Paths.TemplatePath, "settings-users.html")
		footerPage := filepath.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		pkg.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := SettingsResponse{
			IsLoggedIn:         true,
			IsAdmin:            models.CheckifUserIsAnAdmin(db, userID),
			RegisterErrMessage: "",
			HeaderActiveMenu:   "meta",
			SorciaVersion:      conf.Version,
			Users:              users,
			SiteSettings:       GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// PostPasswordRequest struct
type PostPasswordRequest struct {
	Username string `schema:"username"`
	Password string `schema:"password"`
}

// SettingsPostPassword ...
func SettingsPostPassword(w http.ResponseWriter, r *http.Request, db *sql.DB, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")

		// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &pkg.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			pkg.CheckError("Error on json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		postPasswordRequest := &PostPasswordRequest{}
		err := decoder.Decode(postPasswordRequest, r.PostForm)
		pkg.CheckError("Error on post password decoder", err)

		username := models.GetUsernameFromToken(db, token)

		// Generate password hash using bcrypt
		passwordHash, err := HashPassword(postPasswordRequest.Password)
		pkg.CheckError("Error on password hash", err)

		// Generate JWT token using the hash password above
		jwtToken, err := GenerateJWTToken(passwordHash)
		pkg.CheckError("Error on generating jwt token", err)

		resetPass := models.ResetUserPasswordbyUsernameStruct{
			PasswordHash: passwordHash,
			JwtToken:     jwtToken,
			Username:     username,
		}
		models.ResetUserPasswordbyUsername(db, resetPass)
		http.Redirect(w, r, "/meta", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// SettingsPostSiteSettings ...
func SettingsPostSiteSettings(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {

		siteTitle := r.FormValue("title")
		siteStyle := r.FormValue("style")

		gotFavicon, faviconPath := faviconUpload(w, r, db, conf.Paths.UploadAssetPath)
		gotLogo, logoPath, logoWidth, logoHeight := logoUpload(w, r, db, conf.Paths.UploadAssetPath)

		if siteTitle == "" && siteStyle == "" && !gotFavicon && !gotLogo {
			http.Redirect(w, r, "/settings", http.StatusFound)
			return
		}

		if !models.CheckIFSiteSettingsExists(db) {
			css := models.CreateSiteSettingsStruct{
				Title:      siteTitle,
				Favicon:    faviconPath,
				Logo:       logoPath,
				LogoWidth:  logoWidth,
				LogoHeight: logoHeight,
				Style:      siteStyle,
			}
			models.InsertSiteSettings(db, css)

			http.Redirect(w, r, "/settings", http.StatusFound)
			return
		}

		if siteTitle != "" {
			models.UpdateSiteTitle(db, siteTitle)
		}

		if siteStyle != "" {
			models.UpdateSiteStyle(db, siteStyle)
		}

		if gotFavicon {
			models.UpdateSiteFavicon(db, faviconPath)
		}

		if gotLogo {
			models.UpdateSiteLogo(db, logoPath, logoWidth, logoHeight)
		}

		http.Redirect(w, r, "/meta", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func faviconUpload(w http.ResponseWriter, r *http.Request, db *sql.DB, uploadAssetPath string) (bool, string) {
	r.ParseMultipartForm(2)

	file, hdlr, err := r.FormFile("favicon")
	if err != nil {
		return false, ""
	}
	defer file.Close()

	contentType := hdlr.Header.Get("Content-Type")

	if contentType == "image/ico" || contentType == "image/png" || contentType == "image/jpeg" {

		oldFavicon := models.GetSiteFavicon(db)
		if oldFavicon != "" {
			err = os.Remove(oldFavicon)
			pkg.CheckError("Error on removing old favicon", err)
		}

		ext := strings.Split(contentType, "image/")[1]

		filePath := filepath.Join(uploadAssetPath, "favicon."+ext)

		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
		pkg.CheckError("Error on opening favicon path", err)
		defer f.Close()

		io.Copy(f, file)

		return true, filePath
	}

	return false, ""
}

func logoUpload(w http.ResponseWriter, r *http.Request, db *sql.DB, uploadAssetPath string) (bool, string, string, string) {
	r.ParseMultipartForm(10)

	file, hdlr, err := r.FormFile("logo")
	if err != nil {
		return false, "", "", ""
	}
	defer file.Close()

	contentType := hdlr.Header.Get("Content-Type")

	if contentType == "image/svg+xml" || contentType == "image/png" || contentType == "image/jpeg" {

		oldLogo := models.GetSiteLogo(db)
		if oldLogo != "" {
			err = os.Remove(oldLogo)
			pkg.CheckError("Error on removing old logo", err)
		}

		ext := strings.Split(contentType, "image/")[1]

		if ext == "svg+xml" {
			ext = "svg"
		}

		filePath := filepath.Join(uploadAssetPath, "logo."+ext)

		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
		pkg.CheckError("Error on opening favicon path", err)
		defer f.Close()

		io.Copy(f, file)

		getFile, err := os.Open(filePath)
		pkg.CheckError("Error on opening logo upload file", err)
		defer getFile.Close()

		var logoWidth, logoHeight string

		if ext != "svg" {
			image, _, err := image.DecodeConfig(getFile)
			pkg.CheckError("Error on DecodeConfig in logoUpload function", err)
			logoWidth = strconv.Itoa(image.Width)
			logoHeight = strconv.Itoa(image.Height)
		}

		return true, filePath, logoWidth, logoHeight
	}

	return false, "", "", ""
}
