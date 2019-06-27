package main

import (
	"database/sql"
	"fmt"
	"path"

	cError "sorcia/error"
	"sorcia/settings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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

	// Open postgres database
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", conf.Postgres.Username, conf.Postgres.Password, conf.Postgres.Hostname, conf.Postgres.Port, conf.Postgres.Name, conf.Postgres.SSLMode)
	db, err := sql.Open("postgres", connStr)
	cError.CheckError(err)
	defer db.Close()

	// Gin handlers
	r.GET("/", Home)
	r.GET("/login", Login)
	r.GET("/host", GetHostAddress)

	// Listen and serve on 1937
	r.Run(fmt.Sprintf(":%s", conf.Server.HTTPPort))
}

// Home ...
func Home(c *gin.Context) {
	c.HTML(200, "index.html", "")
}

// Login ...
func Login(c *gin.Context) {
	c.HTML(200, "login.html", "")
}

// GetHostAddress returns the URL address
func GetHostAddress(c *gin.Context) {
	c.String(200, fmt.Sprintf("%s", c.Request.Host))
}
