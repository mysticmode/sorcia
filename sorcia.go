package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	r.GET("/~:username", GetHome)
	r.GET("/~:username/:reponame", GetRepo)
	r.GET("/host", GetHostAddress)

	// Git http backend service handlers
	r.POST("/~:username/:reponame/git-:rpc", PostServiceRPC)
	r.GET("/~:username/:reponame/info/refs", GetInfoRefs)
	r.GET("/~:username/:reponame/HEAD", GetHEADFile)
	r.GET("/~:username/:reponame/objects/:regex", GetGitRegexRequestHandler)

	// Listen and serve on 1937
	r.Run(fmt.Sprintf(":%s", conf.Server.HTTPPort))
}

// GetGitRegexRequestHandler ...
func GetGitRegexRequestHandler(c *gin.Context) {
	regex := c.Param("regex")
	fmt.Println(regex)
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

		c.HTML(http.StatusOK, "create-repo.html", gin.H{
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

// GetRepo ...
func GetRepo(c *gin.Context) {
	username := c.Param("username")
	reponame := c.Param("reponame")

	db, ok := c.MustGet("db").(*sql.DB)
	if !ok {
		fmt.Println("Middleware db error")
	}

	rts := model.RepoTypeStruct{
		Username: username,
		Reponame: reponame,
	}

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		c.HTML(http.StatusNotFound, "", "")
		return
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		c.HTML(http.StatusOK, "repo-summary.html", "")
	} else {
		userPresent, ok := c.MustGet("userPresent").(bool)
		if !ok {
			fmt.Println("Middleware user error")
		}

		if userPresent {
			token, _ := c.Cookie("sorcia-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				c.HTML(http.StatusOK, "repo-summary.html", "")
			} else {
				c.HTML(http.StatusNotFound, "", "")
			}
		} else {
			c.Redirect(http.StatusMovedPermanently, "/")
		}
	}
}

// GitHandlerReqStruct struct
type GitHandlerReqStruct struct {
	c    *gin.Context
	RPC  string
	Dir  string
	File string
}

func applyGitHandlerReq(c *gin.Context, rpc string) *GitHandlerReqStruct {
	username := c.Param("username")
	reponame := c.Param("reponame")

	// Get config values
	conf := setting.GetConf()

	repoDir := path.Join(conf.Paths.DataPath, "repositories"+"/"+username+"/"+reponame)

	URLPath := c.Request.URL.Path
	file := strings.Split(URLPath, "/~"+username+"/"+reponame)[1]

	return &GitHandlerReqStruct{
		c:    c,
		RPC:  rpc,
		Dir:  repoDir,
		File: file,
	}
}

func getServiceType(c *gin.Context) string {
	q := c.Request.URL.Query()
	serviceType := q["service"][0]
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.TrimPrefix(serviceType, "git-")
}

func gitCommand(dir string, args ...string) []byte {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	errorhandler.CheckError(err)

	return out
}

func updateServerInfo(dir string) []byte {
	return gitCommand(dir, "update-server-info")
}

func (ghrs *GitHandlerReqStruct) sendFile(contentType string) {
	reqFile := path.Join(ghrs.Dir, ghrs.File)
	fi, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		ghrs.c.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	ghrs.c.Writer.Header().Set("Content-Type", contentType)
	ghrs.c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	ghrs.c.Writer.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	ghrs.c.File(reqFile)
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

func packetFlush() []byte {
	return []byte("0000")
}

func hdrNocache(c *gin.Context) {
	c.Writer.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.Writer.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(c *gin.Context) {
	now := time.Now().Unix()
	expires := now + 31536000
	c.Writer.Header().Set("Date", fmt.Sprintf("%d", now))
	c.Writer.Header().Set("Expires", fmt.Sprintf("%d", expires))
	c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
}

// PostServiceRPC ...
func PostServiceRPC(c *gin.Context) {
	username := c.Param("username")
	reponame := c.Param("reponame")
	rpc := c.Param("rpc")

	if rpc != "upload-pack" && rpc != "receive-pack" {
		return
	}

	c.Writer.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	c.Writer.Header().Set("Connection", "Keep-Alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(http.StatusOK)

	reqBody := c.Request.Body
	var err error

	// Handle GZip
	if c.Request.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			errorhandler.CheckError(err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Get config values
	conf := setting.GetConf()

	repoDir := (path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+reponame))

	cmd := exec.Command("git", rpc, "--stateless-rpc", repoDir)

	var stderr bytes.Buffer

	cmd.Dir = repoDir
	cmd.Stdout = c.Writer
	cmd.Stderr = &stderr
	cmd.Stdin = reqBody
	if err := cmd.Run(); err != nil {
		errorhandler.CheckError(err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// GetInfoRefs ...
func GetInfoRefs(c *gin.Context) {
	hdrNocache(c)

	service := getServiceType(c)
	args := []string{service, "--stateless-rpc", "--advertise-refs", "."}

	ghrs := applyGitHandlerReq(c, "")

	if service != "upload-pack" && service != "receive-pack" {
		updateServerInfo(ghrs.Dir)
		ghrs.sendFile("text/plain; charset=utf-8")
		return
	}

	refs := gitCommand(ghrs.Dir, args...)
	c.Writer.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(packetWrite("# service=git-" + service + "\n"))
	c.Writer.Write(packetFlush())
	c.Writer.Write(refs)
}

// GetInfoPacks ...
func (ghrs *GitHandlerReqStruct) GetInfoPacks(c *gin.Context) {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("text/plain; charset=utf-8")
}

// GetLooseObject ...
func (ghrs *GitHandlerReqStruct) GetLooseObject(c *gin.Context) {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-loose-object")
}

// GetPackFile ...
func (ghrs *GitHandlerReqStruct) GetPackFile(c *gin.Context) {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-packed-objects")
}

// GetIdxFile ...
func (ghrs *GitHandlerReqStruct) GetIdxFile() {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-packed-objects-toc")
}

// GetTextFile ...
func (ghrs *GitHandlerReqStruct) GetTextFile() {
	hdrNocache(ghrs.c)
	ghrs.sendFile("text/plain")
}

// GetHEADFile ...
func GetHEADFile(c *gin.Context) {
	hdrNocache(c)
	ghrs := applyGitHandlerReq(c, "")
	ghrs.sendFile("text/plain")
}
