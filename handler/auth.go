package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	errorhandler "sorcia/error"
	"sorcia/model"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/schema"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWTToken(passwordHash string) (string, error) {
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
	IsHeaderLogin      bool
	HeaderActiveMenu   string
	LoginErrMessage    string
	RegisterErrMessage string
}

// GetLogin ...
func GetLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, templatePath string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		layoutPage := path.Join(templatePath, "templates", "layout.tmpl")
		headerPage := path.Join(templatePath, "templates", "header.tmpl")
		loginPage := path.Join(templatePath, "templates", "login.tmpl")
		footerPage := path.Join(templatePath, "templates", "footer.tmpl")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := LoginPageResponse{
			IsHeaderLogin:      true,
			HeaderActiveMenu:   "",
			LoginErrMessage:    "",
			RegisterErrMessage: "",
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
func PostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, dataPath, templatePath string, decoder *schema.Decoder) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &errorhandler.ErrorResponse{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	isRegisterForm := r.FormValue("register")
	if isRegisterForm == "1" {
		postRegister(w, r, db, dataPath, templatePath, decoder)
		return
	}

	var loginRequest = &LoginRequest{}
	err := decoder.Decode(loginRequest, r.PostForm)
	errorhandler.CheckError(err)

	sphjwt := model.SelectPasswordHashAndJWTTokenStruct{
		Username: loginRequest.Username,
	}
	sphjwtr := model.SelectPasswordHashAndJWTToken(db, sphjwt)

	if isPasswordValid := checkPasswordHash(loginRequest.Password, sphjwtr.PasswordHash); isPasswordValid == true {
		isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
		errorhandler.CheckError(err)

		if isTokenValid == true {
			// Set cookie
			now := time.Now()
			duration := now.Add(365 * 24 * time.Hour).Sub(now)
			maxAge := int(duration.Seconds())
			c := &http.Cookie{Name: "sorcia-token", Value: sphjwtr.Token, Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: maxAge}
			http.SetCookie(w, c)

			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			invalidLoginCredentials(w, r, templatePath)
		}
	} else {
		invalidLoginCredentials(w, r, templatePath)
	}
}

func invalidLoginCredentials(w http.ResponseWriter, r *http.Request, templatePath string) {
	layoutPage := path.Join(templatePath, "templates", "layout.tmpl")
	headerPage := path.Join(templatePath, "templates", "header.tmpl")
	loginPage := path.Join(templatePath, "templates", "login.tmpl")
	footerPage := path.Join(templatePath, "templates", "footer.tmpl")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage, footerPage)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	data := LoginPageResponse{
		IsHeaderLogin:      true,
		HeaderActiveMenu:   "",
		LoginErrMessage:    "Your username or password is incorrect.",
		RegisterErrMessage: "",
	}

	tmpl.ExecuteTemplate(w, "layout", data)
}

func isAlnumOrHyphen(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// RegisterRequest struct
type RegisterRequest struct {
	Username string `schema:"username"`
	Email    string `schema:"email"`
	Password string `schema:"password"`
	Register string `schema:"register"`
}

// PostRegister ...
func postRegister(w http.ResponseWriter, r *http.Request, db *sql.DB, dataPath string, templatePath string, decoder *schema.Decoder) {
	var registerRequest = &RegisterRequest{}
	err := decoder.Decode(registerRequest, r.PostForm)
	errorhandler.CheckError(err)

	if registerRequest.Username != "" && registerRequest.Email != "" && registerRequest.Password != "" {
		// Generate password hash using bcrypt
		passwordHash, err := hashPassword(registerRequest.Password)
		errorhandler.CheckError(err)

		// Generate JWT token using the hash password above
		token, err := generateJWTToken(passwordHash)
		errorhandler.CheckError(err)

		s := registerRequest.Username

		if len(s) > 39 || len(s) < 1 {
			layoutPage := path.Join(templatePath, "templates", "layout.tmpl")
			headerPage := path.Join(templatePath, "templates", "header.tmpl")
			loginPage := path.Join(templatePath, "templates", "login.tmpl")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage)
			errorhandler.CheckError(err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsHeaderLogin:      true,
				LoginErrMessage:    "",
				RegisterErrMessage: "Username is too long (maximum is 39 characters).",
			}

			tmpl.ExecuteTemplate(w, "layout", data)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			tmpl.Execute(w, data)
			return
		} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !isAlnumOrHyphen(s) {
			layoutPage := path.Join(templatePath, "templates", "layout.tmpl")
			headerPage := path.Join(templatePath, "templates", "header.tmpl")
			loginPage := path.Join(templatePath, "templates", "login.tmpl")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, loginPage)
			errorhandler.CheckError(err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				IsHeaderLogin:      true,
				HeaderActiveMenu:   "",
				LoginErrMessage:    "",
				RegisterErrMessage: "Username may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}

		rr := model.CreateAccountStruct{
			Username:     registerRequest.Username,
			Email:        registerRequest.Email,
			PasswordHash: passwordHash,
			Token:        token,
			IsAdmin:      1,
		}

		model.InsertAccount(db, rr)

		// Create repositories directory
		// 0755 - The owner can read, write, execute. Everyone else can read and execute but not modify the file.
		repoDir := path.Join(dataPath, "repositories/"+"+"+registerRequest.Username)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			os.MkdirAll(repoDir, 0755)
		}

		// Set cookie
		now := time.Now()
		duration := now.Add(365 * 24 * time.Hour).Sub(now)
		maxAge := int(duration.Seconds())
		c := &http.Cookie{Name: "sorcia-token", Value: token, Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: maxAge}
		http.SetCookie(w, c)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// GetLogout ...
func GetLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the cookie
	c := &http.Cookie{Name: "sorcia-token", Value: "", Path: "/", Domain: strings.Split(r.Host, ":")[0], MaxAge: -1}
	http.SetCookie(w, c)

	http.Redirect(w, r, "/login", http.StatusFound)
}
