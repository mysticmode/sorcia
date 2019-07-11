package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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
	r.GET("/login", handler.GetLogin)
	r.POST("/login", handler.PostLogin)
	r.GET("/logout", handler.GetLogout)
	r.POST("/register", handler.PostRegister)
	r.GET("/create", handler.GetCreateRepo)
	r.POST("/create", handler.PostCreateRepo)
	r.GET("/~:username", GetHome)
	r.GET("/~:username/:reponame", handler.GetRepo)
	r.GET("/host", GetHostAddress)

	// Git http backend service handlers
	r.POST("/~:username/:reponame/git-:rpc", handler.PostServiceRPC)
	r.GET("/~:username/:reponame/info/refs", handler.GetInfoRefs)
	r.GET("/~:username/:reponame/HEAD", handler.GetHEADFile)
	r.GET("/~:username/:reponame/objects/:regex1/:regex2", handler.GetGitRegexRequestHandler)

	// Listen and serve on 1937
	r.Run(fmt.Sprintf(":%s", conf.Server.HTTPPort))
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

		username = "~" + username

		repos := model.GetReposFromUserID(db, userID)

		c.HTML(200, "index.html", gin.H{
			"username": username,
			"repos":    repos,
		})
	} else {
		c.Redirect(http.StatusMovedPermanently, "/login")
	}
}

// GetHostAddress returns the URL address
func GetHostAddress(c *gin.Context) {
	c.String(200, fmt.Sprintf("%s", c.Request.Host))
}
