package main

import "github.com/gin-gonic/gin"

func main() {
	// Gin initiate
	r := gin.Default()

	// HTML rendering
	r.LoadHTMLGlob("templates/*")

	// Serve static files
	r.Static("/public", "./public")

	// Gin handlers
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", "")
	})

	// Listen and serve on 1937
	r.Run(":1937")
}
