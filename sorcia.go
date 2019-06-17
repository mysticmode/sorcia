package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	// Gin initiate
	r := gin.Default()

	// HTML rendering
	r.LoadHTMLGlob("templates/*")

	// Serve static files
	r.Static("/public", "./public")

	// Gin handlers
	r.GET("/", Home)
	r.GET("/host", GetHostAddress)

	// Listen and serve on 1937
	r.Run(":1937")
}

// Home ...
func Home(c *gin.Context) {
	c.HTML(200, "repo-tree.html", "")
}

// GetHostAddress returns the URL address
func GetHostAddress(c *gin.Context) {
	c.String(200, fmt.Sprintf("%s", c.Request.Host))
}
