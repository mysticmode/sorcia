package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
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

// GetLogin ...
func GetLogin(c *gin.Context) {
	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"loginErrMessage": "",
		})
	}
}

// LoginRequest struct
type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// PostLogin ...
func PostLogin(c *gin.Context) {
	var form LoginRequest

	if err := c.Bind(&form); err == nil {
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		// Sorcia username identity
		username := form.Username

		sphjwt := model.SelectPasswordHashAndJWTTokenStruct{
			Username: username,
		}
		sphjwtr := model.SelectPasswordHashAndJWTToken(db, sphjwt)

		if isPasswordValid := checkPasswordHash(form.Password, sphjwtr.PasswordHash); isPasswordValid == true {
			isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
			errorhandler.CheckError(err)

			if isTokenValid == true {
				c.SetCookie("sorcia-token", sphjwtr.Token, 0, "/", strings.Split(c.Request.Host, ":")[0], false, true)

				c.Redirect(http.StatusMovedPermanently, "/")
			} else {
				c.HTML(http.StatusOK, "login.html", gin.H{
					"loginErrMessage": "Your username or password is incorrect.",
				})
			}
		} else {
			c.HTML(http.StatusOK, "login.html", gin.H{
				"loginErrMessage": "Your username or password is incorrect.",
			})
		}
	} else {
		errorResponse := &errorhandler.ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
}

// RegisterRequest struct
type RegisterRequest struct {
	Username string `form:"username" binding:"required"`
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}

// PostRegister ...
func PostRegister(c *gin.Context) {
	var form RegisterRequest

	if err := c.Bind(&form); err == nil {
		// Generate password hash using bcrypt
		passwordHash, err := hashPassword(form.Password)
		errorhandler.CheckError(err)

		// Generate JWT token using the hash password above
		token, err := generateJWTToken(passwordHash)
		errorhandler.CheckError(err)

		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		usernameConvention := "^[a-zA-Z0-9_]*$"

		if re := regexp.MustCompile(usernameConvention); !re.MatchString(form.Username) {
			c.HTML(http.StatusOK, "login.html", gin.H{
				"registerErrMessage": "Username is invalid. Supports only alphanumeric and underscore characters.",
			})

			return
		}

		// Sorcia username identity
		username := form.Username

		rr := model.CreateAccountStruct{
			Username:     username,
			Email:        form.Email,
			PasswordHash: passwordHash,
			Token:        token,
			IsAdmin:      1,
		}

		model.InsertAccount(db, rr)

		// Get config values
		conf := setting.GetConf()

		// Create repositories directory
		// 0755 - The owner can read, write, execute. Everyone else can read and execute but not modify the file.
		repoDir := path.Join(conf.Paths.DataPath, "repositories/"+username)
		if _, err := os.Stat(repoDir); os.IsNotExist(err) {
			os.MkdirAll(repoDir, 0755)
		}

		c.SetCookie("sorcia-token", token, 5259492, "/", strings.Split(c.Request.Host, ":")[0], true, true)

		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		errorResponse := &errorhandler.ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
}

// GetLogout ...
func GetLogout(c *gin.Context) {
	// Clear the cookie
	c.SetCookie("sorcia-token", "", -1, "/", "", true, true)

	c.Redirect(http.StatusTemporaryRedirect, "/login")
}
