package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"
	"time"

	"sorcia/models"
	"sorcia/pkg"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/schema"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword ...
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash ...
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWTToken ...
func GenerateJWTToken(passwordHash string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	tokenString, err := token.SignedString([]byte(passwordHash))
	return tokenString, err
}

func validateJWTToken(tokenString string, passwordHash string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(passwordHash), nil
	})

	return token.Valid, err
}

// LoginPageResponse struct
type LoginPageResponse struct {
	IsLoggedIn         bool
	ShowLoginMenu      bool
	HeaderActiveMenu   string
	SorciaVersion      string
	IsShowSignUp       bool
	LoginErrMessage    string
	RegisterErrMessage string
	SiteSettings       SiteSettings
}

// GetLogin ...
func GetLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		layoutPage := path.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := path.Join(conf.Paths.TemplatePath, "header.html")
		loginPage := path.Join(conf.Paths.TemplatePath, "login.html")
		footerPage := path.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
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
			RegisterErrMessage: "",
			SiteSettings:       GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	}
}

// LoginRequest struct
type LoginRequest struct {
	Username string `schema:"username"`
	Password string `schema:"password"`
}

// PostLogin ...
func PostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct, decoder *schema.Decoder) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &pkg.Response{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		pkg.CheckError("Error on post login json marshal", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	isRegisterForm := r.FormValue("register")
	if isRegisterForm == "1" {
		postRegister(w, r, db, conf, decoder)
		return
	}

	var loginRequest = &LoginRequest{}
	err := decoder.Decode(loginRequest, r.PostForm)
	pkg.CheckError("Error on post login decoder", err)

	sphjwt := models.SelectPasswordHashAndJWTTokenStruct{
		Username: loginRequest.Username,
	}
	sphjwtr := models.SelectPasswordHashAndJWTToken(db, sphjwt)

	if isPasswordValid := CheckPasswordHash(loginRequest.Password, sphjwtr.PasswordHash); isPasswordValid == true {
		isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
		pkg.CheckError("Error on validating jwt token", err)

		if isTokenValid == true {
			// Set cookie
			now := time.Now()
			duration := now.Add(365 * 24 * time.Hour).Sub(now)
			maxAge := int(duration.Seconds())
			c := &http.Cookie{Name: "sorcia-token", Value: sphjwtr.Token, Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: maxAge}
			http.SetCookie(w, c)

			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			invalidLoginCredentials(w, r, db, conf)
		}
	} else {
		invalidLoginCredentials(w, r, db, conf)
	}
}

func invalidLoginCredentials(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	layoutPage := path.Join(conf.Paths.TemplatePath, "layout.html")
	headerPage := path.Join(conf.Paths.TemplatePath, "header.html")
	loginPage := path.Join(conf.Paths.TemplatePath, "login.html")
	footerPage := path.Join(conf.Paths.TemplatePath, "footer.html")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
	pkg.CheckError("Error on template parse", err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	data := LoginPageResponse{
		IsLoggedIn:         false,
		ShowLoginMenu:      false,
		HeaderActiveMenu:   "",
		SorciaVersion:      conf.Version,
		LoginErrMessage:    "Your username or password is incorrect.",
		RegisterErrMessage: "",
		SiteSettings:       GetSiteSettings(db, conf),
	}

	tmpl.ExecuteTemplate(w, "layout", data)
}

// RegisterRequest struct
type RegisterRequest struct {
	Username string `schema:"username"`
	Password string `schema:"password"`
	Register string `schema:"register"`
}

// PostRegister ...
func postRegister(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct, decoder *schema.Decoder) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &pkg.Response{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		pkg.CheckError("Error on post register json marshal", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	var registerRequest = &RegisterRequest{}
	err := decoder.Decode(registerRequest, r.PostForm)
	pkg.CheckError("Error on post register decoder", err)

	// Generate password hash using bcrypt
	passwordHash, err := HashPassword(registerRequest.Password)
	pkg.CheckError("Error on post register hash password", err)

	// Generate JWT token using the hash password above
	token, err := GenerateJWTToken(passwordHash)
	pkg.CheckError("Error on post register generate jwt token", err)

	s := registerRequest.Username

	if len(s) > 39 || len(s) < 1 {
		layoutPage := path.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := path.Join(conf.Paths.TemplatePath, "header.html")
		loginPage := path.Join(conf.Paths.TemplatePath, "login.html")
		footerPage := path.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
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
		layoutPage := path.Join(conf.Paths.TemplatePath, "layout.html")
		headerPage := path.Join(conf.Paths.TemplatePath, "header.html")
		loginPage := path.Join(conf.Paths.TemplatePath, "login.html")
		footerPage := path.Join(conf.Paths.TemplatePath, "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
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

	rr := models.CreateAccountStruct{
		Username:      registerRequest.Username,
		PasswordHash:  passwordHash,
		Token:         token,
		CanCreateRepo: 1,
		IsAdmin:       1,
	}

	models.InsertAccount(db, rr)

	// Set cookie
	now := time.Now()
	duration := now.Add(365 * 24 * time.Hour).Sub(now)
	maxAge := int(duration.Seconds())
	c := &http.Cookie{Name: "sorcia-token", Value: token, Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: maxAge}
	http.SetCookie(w, c)

	http.Redirect(w, r, "/", http.StatusFound)
}

// GetLogout ...
func GetLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the cookie
	c := &http.Cookie{Name: "sorcia-token", Value: "", Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: -1}
	http.SetCookie(w, c)

	http.Redirect(w, r, "/login", http.StatusFound)
}
