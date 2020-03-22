package handler

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
	"path"
	"path/filepath"
	"strconv"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"
	"sorcia/util"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// MetaResponse struct
type MetaResponse struct {
	IsLoggedIn         bool
	IsAdmin            bool
	HeaderActiveMenu   string
	SorciaVersion      string
	Username           string
	Email              string
	Users              model.Users
	RegisterErrMessage string
	SiteSettings       util.SiteSettings
}

// GetMeta ...
func GetMeta(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		username := model.GetUsernameFromToken(db, token)

		userID := model.GetUserIDFromToken(db, token)

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
			IsAdmin:          model.CheckifUserIsAnAdmin(db, userID),
			HeaderActiveMenu: "meta",
			SorciaVersion:    conf.Version,
			Username:         username,
			SiteSettings:     util.GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// MetaKeysResponse struct
type MetaKeysResponse struct {
	IsLoggedIn       bool
	IsAdmin          bool
	HeaderActiveMenu string
	SorciaVersion    string
	SSHKeys          *model.SSHKeysResponse
	SiteSettings     util.SiteSettings
}

// GetMetaKeys ...
func GetMetaKeys(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
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
			IsAdmin:          model.CheckifUserIsAnAdmin(db, userID),
			HeaderActiveMenu: "meta",
			SorciaVersion:    conf.Version,
			SSHKeys:          sshKeys,
			SiteSettings:     util.GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// DeleteMetaKey ...
func DeleteMetaKey(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	keyID := vars["keyID"]

	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		i, err := strconv.Atoi(keyID)
		errorhandler.CheckError("Error on converting SSH key id(string) to int on delete meta keys", err)

		model.DeleteMetaKeyByID(db, i)
		http.Redirect(w, r, "/meta/keys", http.StatusFound)
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

		var createAuthKeyRequest = &CreateAuthKeyRequest{}
		err := decoder.Decode(createAuthKeyRequest, r.PostForm)
		errorhandler.CheckError("Error on auth key decode", err)

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
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// PostUserRequest struct
type PostUserRequest struct {
	Username      string `schema:"username"`
	Password      string `schema:"password"`
	CanCreateRepo string `schema:"createrepo"`
}

// PostUser ...
func PostUser(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
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

		var postUserRequest = &PostUserRequest{}
		err := decoder.Decode(postUserRequest, r.PostForm)
		errorhandler.CheckError("Error on meta post user", err)

		// Generate password hash using bcrypt
		passwordHash, err := HashPassword(postUserRequest.Password)
		errorhandler.CheckError("Error on post register hash password", err)

		// Generate JWT token using the hash password above
		token, err := GenerateJWTToken(passwordHash)
		errorhandler.CheckError("Error on post register generate jwt token", err)

		s := postUserRequest.Username

		if len(s) > 39 || len(s) < 1 {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			metaUsersPage := path.Join("./templates", "meta-users.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, metaUsersPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsLoggedIn:         false,
				ShowLoginMenu:      false,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				IsShowSignUp:       !model.CheckIfFirstUserExists(db),
				LoginErrMessage:    "",
				RegisterErrMessage: "Username is too long (maximum is 39 characters).",
				SiteSettings:       util.GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !util.IsAlnumOrHyphen(s) {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			metaUsersPage := path.Join("./templates", "meta-users.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, metaUsersPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsLoggedIn:         false,
				ShowLoginMenu:      false,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				IsShowSignUp:       !model.CheckIfFirstUserExists(db),
				LoginErrMessage:    "",
				RegisterErrMessage: "Username may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
				SiteSettings:       util.GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}

		canCreateRepo := 0
		if postUserRequest.CanCreateRepo != "" {
			canCreateRepo = 1
		}

		rr := model.CreateAccountStruct{
			Username:      postUserRequest.Username,
			PasswordHash:  passwordHash,
			Token:         token,
			CanCreateRepo: canCreateRepo,
			IsAdmin:       0,
		}

		model.InsertAccount(db, rr)

		http.Redirect(w, r, "/meta/users", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// GetMetaUsers ...
func GetMetaUsers(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		users := model.GetAllUsers(db)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta-users.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := MetaResponse{
			IsLoggedIn:         true,
			IsAdmin:            model.CheckifUserIsAnAdmin(db, userID),
			RegisterErrMessage: "",
			HeaderActiveMenu:   "meta",
			SorciaVersion:      conf.Version,
			Users:              users,
			SiteSettings:       util.GetSiteSettings(db, conf),
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

// MetaPostPassword ...
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
		jwtToken, err := GenerateJWTToken(passwordHash)
		errorhandler.CheckError("Error on generating jwt token", err)

		resetPass := model.ResetUserPasswordbyUsernameStruct{
			PasswordHash: passwordHash,
			JwtToken:     jwtToken,
			Username:     username,
		}
		model.ResetUserPasswordbyUsername(db, resetPass)
		http.Redirect(w, r, "/meta", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// MetaPostSiteSettings ...
func MetaPostSiteSettings(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {

		siteTitle := r.FormValue("title")
		siteStyle := r.FormValue("style")

		gotFavicon, faviconPath := faviconUpload(w, r, db, conf.Paths.UploadAssetPath)
		gotLogo, logoPath, logoWidth, logoHeight := logoUpload(w, r, db, conf.Paths.UploadAssetPath)

		if siteTitle == "" && siteStyle == "" && !gotFavicon && !gotLogo {
			http.Redirect(w, r, "/meta", http.StatusFound)
			return
		}

		if !model.CheckIFSiteSettingsExists(db) {
			css := model.CreateSiteSettingsStruct{
				Title:      siteTitle,
				Favicon:    faviconPath,
				Logo:       logoPath,
				LogoWidth:  logoWidth,
				LogoHeight: logoHeight,
				Style:      siteStyle,
			}
			model.InsertSiteSettings(db, css)

			http.Redirect(w, r, "/meta", http.StatusFound)
			return
		}

		if siteTitle != "" {
			model.UpdateSiteTitle(db, siteTitle)
		}

		if siteStyle != "" {
			model.UpdateSiteStyle(db, siteStyle)
		}

		if gotFavicon {
			model.UpdateSiteFavicon(db, faviconPath)
		}

		if gotLogo {
			model.UpdateSiteLogo(db, logoPath, logoWidth, logoHeight)
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

		oldFavicon := model.GetSiteFavicon(db)
		if oldFavicon != "" {
			err = os.Remove(oldFavicon)
			errorhandler.CheckError("Error on removing old favicon", err)
		}

		ext := strings.Split(contentType, "image/")[1]

		filePath := filepath.Join(uploadAssetPath, "favicon."+ext)

		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
		errorhandler.CheckError("Error on opening favicon path", err)
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

		oldLogo := model.GetSiteLogo(db)
		if oldLogo != "" {
			err = os.Remove(oldLogo)
			errorhandler.CheckError("Error on removing old logo", err)
		}

		ext := strings.Split(contentType, "image/")[1]

		if ext == "svg+xml" {
			ext = "svg"
		}

		filePath := filepath.Join(uploadAssetPath, "logo."+ext)

		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
		errorhandler.CheckError("Error on opening favicon path", err)
		defer f.Close()

		io.Copy(f, file)

		getFile, err := os.Open(filePath)
		errorhandler.CheckError("Error on opening logo upload file", err)
		defer getFile.Close()

		var logoWidth, logoHeight string

		if ext != "svg" {
			image, _, err := image.DecodeConfig(getFile)
			errorhandler.CheckError("Error on DecodeConfig in logoUpload function", err)
			logoWidth = strconv.Itoa(image.Width)
			logoHeight = strconv.Itoa(image.Height)
		}

		return true, filePath, logoWidth, logoHeight
	}

	return false, "", "", ""
}
