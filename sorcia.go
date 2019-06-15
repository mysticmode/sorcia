package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/public", "./public")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", "")
	})

	r.Run(":1937") // listen and serve on 0.0.0.0:1937
}
