package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Gin initiate
	r := gin.Default()

	// Get config values
	conf := setting.GetConf()

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
	errorhandler.CheckError(err)
	defer db.Close()

	model.CreateAccount(db)
	model.CreateRepo(db)

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
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)

		repos := model.GetReposFromUserID(db, userID)

		c.HTML(200, "index.html", gin.H{
			"username": username,
			"repos":    repos,
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

		// Sorcia username identity
		username := "~" + form.Username

		sphjwt := model.SelectPasswordHashAndJWTTokenStruct{
			Username: username,
		}
		sphjwtr := model.SelectPasswordHashAndJWTToken(db, sphjwt)

		if isPasswordValid := checkPasswordHash(form.Password, sphjwtr.PasswordHash); isPasswordValid == true {
			isTokenValid, err := validateJWTToken(sphjwtr.Token, sphjwtr.PasswordHash)
			errorhandler.CheckError(err)

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
		username := "~" + form.Username

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
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		token, _ := c.Cookie("sorcia-token")

		username := model.GetUsernameFromToken(db, token)

		c.HTML(http.StatusOK, "create-model.html", gin.H{
			"username": username,
		})
	} else {
		c.Redirect(http.StatusMovedPermanently, "/login")
	}
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `form:"name" binding:"required"`
	Description string `form:"description" binding:"required"`
	IsPrivate   string `form:"is_private" binding:"required"`
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

		userID := model.GetUserIDFromToken(db, token)

		var isPrivate int
		if isPrivate = 0; form.IsPrivate == "1" {
			isPrivate = 1
		}

		crs := model.CreateRepoStruct{
			Name:        form.Name,
			Description: form.Description,
			IsPrivate:   isPrivate,
			UserID:      userID,
		}

		model.InsertRepo(db, crs)

		// Get config values
		conf := setting.GetConf()

		username := model.GetUsernameFromToken(db, token)

		// Create Git bare repository
		bareRepoDir := path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+form.Name+".git")

		cmd := exec.Command("git", "init", "--bare", bareRepoDir)
		err := cmd.Run()
		errorhandler.CheckError(err)

		// Clone from the bare repository created above
		repoDir := path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+form.Name)
		cmd = exec.Command("git", "clone", bareRepoDir, repoDir)
		err = cmd.Run()
		errorhandler.CheckError(err)

		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		errorResponse := &ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
}
