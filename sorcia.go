package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	cError "sorcia/error"
	"sorcia/middleware"
	"sorcia/models/auth"
	"sorcia/models/repo"
	"sorcia/settings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Gin initiate
	r := gin.Default()

	// Get config values
	conf := settings.GetConf()

	// HTML rendering
	r.LoadHTMLGlob(path.Join(conf.Paths.TemplatePath, "templates/*"))

	// Serve static files
	r.Static("/public", path.Join(conf.Paths.AssetPath, "public"))

	// Create repositories directory
	// 0755 - The owner can read, write, execute. Everyone else can read and execute but not modify the file.
	os.MkdirAll(path.Join(conf.Paths.DataPath, "repositories"), 0755)

	// Open postgres database
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", conf.Postgres.Username, conf.Postgres.Password, conf.Postgres.Hostname, conf.Postgres.Port, conf.Postgres.Name, conf.Postgres.SSLMode)
	db, err := sql.Open("postgres", connStr)
	cError.CheckError(err)
	defer db.Close()

	auth.CreateAccount(db)
	repo.CreateRepo(db)

	r.Use(
		middleware.CORSMiddleware(),
		middleware.APIMiddleware(db),
		middleware.UserMiddleware(db),
	)

	// Gin handlers
	r.GET("/", GetHome)
	r.GET("/login", GetLogin)
	r.POST("/login", PostLogin)
	r.GET("/logout", GetLogout)
	r.POST("/register", PostRegister)
	r.GET("/create", GetCreateRepo)
	r.POST("/create", PostCreateRepo)
	r.GET("/host", GetHostAddress)

	// Listen and serve on 1937
	r.Run(fmt.Sprintf(":%s", conf.Server.HTTPPort))
}

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

// ErrorResponse struct
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetHome ...
func GetHome(c *gin.Context) {
	db, ok := c.MustGet("db").(*sql.DB)
	if !ok {
		fmt.Println("Middleware db error")
	}

	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		token, _ := c.Cookie("sorcia-token")
		userID := auth.GetUserIDFromToken(db, token)
		username := auth.GetUsernameFromToken(db, token)

		repos := repo.GetReposFromUserID(db, userID)

		c.HTML(200, "index.html", gin.H{
			"username": username,
			"repos": repos,
		})
	} else {
		c.Redirect(http.StatusMovedPermanently, "/login")
	}
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

		sphjwt := auth.SelectPasswordHashAndJWTTokenStruct{
			Username: form.Username,
		}
		sphjwtr := auth.SelectPasswordHashAndJWTToken(db, sphjwt)

		if isPasswordValid := checkPasswordHash(form.Password, sphjwtr.PasswordHash); isPasswordValid == true {
			isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
			cError.CheckError(err)

			if isTokenValid == true {
				fmt.Println(sphjwtr.Token)
				fmt.Println("y")
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
		errorResponse := &ErrorResponse{
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
		cError.CheckError(err)

		// Generate JWT token using the hash password above
		token, err := generateJWTToken(passwordHash)
		cError.CheckError(err)

		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		// Sorcia username identity
		username := "~" + form.Username

		rr := auth.CreateAccountStruct{
			Username:     username,
			Email:        form.Email,
			PasswordHash: passwordHash,
			Token:        token,
			IsAdmin:      1,
		}

		auth.InsertAccount(db, rr)

		c.SetCookie("sorcia-token", token, 0, "/", strings.Split(c.Request.Host, ":")[0], false, true)

		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		errorResponse := &ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
}

// GetLogout ...
func GetLogout(c *gin.Context) {
	// Clear the cookie
	c.SetCookie("sorcia-token", "", -1, "", "", false, true)

	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// GetHostAddress returns the URL address
func GetHostAddress(c *gin.Context) {
	c.String(200, fmt.Sprintf("%s", c.Request.Host))
}

// GetCreateRepo ...
func GetCreateRepo(c *gin.Context) {
	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		c.HTML(http.StatusOK, "create-repo.html", "")
	} else {
		c.Redirect(http.StatusMovedPermanently, "/login")
	}
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `form:"name" binding:"required"`
	Description string `form:"description" binding:"required"`
	Type        string `form:"type" binding:"required"`
}

// PostCreateRepo ...
func PostCreateRepo(c *gin.Context) {
	var form CreateRepoRequest

	if err := c.Bind(&form); err == nil {
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		token, _ := c.Cookie("sorcia-token")

		userID := auth.GetUserIDFromToken(db, token)

		crs := repo.CreateRepoStruct{
			Name:        form.Name,
			Description: form.Description,
			RepoType:    form.Type,
			UserID:      userID,
		}

		repo.InsertRepo(db, crs)

		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		errorResponse := &ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
}
