package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

	"github.com/dgrijalva/jwt-go"
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
	LoginErrMessage    string
	RegisterErrMessage string
}

// GetLogin ...
func GetLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		http.Redirect(w, r, "/login", http.StatusFound)
	} else {
		tmpl := template.Must(template.ParseFiles("./templates/login.html"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := LoginPageResponse{
			LoginErrMessage:    "",
			RegisterErrMessage: "",
		}

		tmpl.Execute(w, data)
	}
}

// LoginRequest struct
type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// PostLogin ...
func PostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	loginRequest := &LoginRequest{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	sphjwt := model.SelectPasswordHashAndJWTTokenStruct{
		Username: loginRequest.Username,
	}
	sphjwtr := model.SelectPasswordHashAndJWTToken(db, sphjwt)

	if isPasswordValid := checkPasswordHash(loginRequest.Password, sphjwtr.PasswordHash); isPasswordValid == true {
		isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
		errorhandler.CheckError(err)

		if isTokenValid == true {
			// Set cookie
			expiration := time.Now().Add(365 * 24 * time.Hour)
			c := &http.Cookie{Name: "sorcia-token", Value: sphjwtr.Token, Path: "/", Domain: strings.Split(r.Host, ":")[0], Expires: expiration}
			http.SetCookie(w, c)

			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			invalidLoginCredentials(w, r)
		}
	} else {
		invalidLoginCredentials(w, r)
	}
}

func invalidLoginCredentials(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./templates/login.html"))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	data := LoginPageResponse{
		LoginErrMessage:    "Your username or password is incorrect.",
		RegisterErrMessage: "",
	}

	tmpl.Execute(w, data)
}

// RegisterRequest struct
type RegisterRequest struct {
	Username string `form:"username" binding:"required"`
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// PostRegister ...
func PostRegister(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	registerRequest := &RegisterRequest{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	if registerRequest.Username != "" && registerRequest.Email != "" && registerRequest.Password != "" {
		// Generate password hash using bcrypt
		passwordHash, err := hashPassword(registerRequest.Password)
		errorhandler.CheckError(err)

		// Generate JWT token using the hash password above
		token, err := generateJWTToken(passwordHash)
		errorhandler.CheckError(err)

		usernameConvention := "^[a-zA-Z0-9_]*$"

		if re := regexp.MustCompile(usernameConvention); !re.MatchString(registerRequest.Username) {
			tmpl := template.Must(template.ParseFiles("./templates/login.html"))

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := LoginPageResponse{
				LoginErrMessage:    "",
				RegisterErrMessage: "Username is invalid. Supports only alphanumeric and underscore characters.",
			}

			tmpl.Execute(w, data)

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

		// Get config values
		conf := setting.GetConf()

		// Create repositories directory
		// 0755 - The owner can read, write, execute. Everyone else can read and execute but not modify the file.
		repoDir := path.Join(conf.Paths.DataPath, "repositories/"+registerRequest.Username)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			os.MkdirAll(repoDir, 0755)
		}

		// Set cookie
		expiration := time.Now().Add(365 * 24 * time.Hour)
		c := &http.Cookie{Name: "sorcia-token", Value: token, Path: "/", Domain: strings.Split(r.Host, ":")[0], Expires: expiration}
		http.SetCookie(w, c)

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// GetLogout ...
func GetLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the cookie
	expiration := time.Now().Add(365 * 24 * time.Hour)
	c := &http.Cookie{Name: "sorcia-token", Value: "", MaxAge: -1, Path: "/", Domain: strings.Split(r.Host, ":")[0], Expires: expiration}
	http.SetCookie(w, c)

	http.Redirect(w, r, "/login", http.StatusFound)
}
