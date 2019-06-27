package middleware

import (
	"database/sql"
	"fmt"

	"sorcia/models/auth"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware ...
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.Request.Header.Get("Origin"); origin != "" {
			c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
		}

		c.Next()
	}
}

// APIMiddleware ...
func APIMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// UserMiddleware ...
func UserMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		token, _ := c.Cookie("sorcia-token")
		fmt.Println(token)
		userID := auth.GetUserIDFromToken(db, token)

		userPresent := false
		if userID != 0 {
			userPresent = true
		}

		c.Set("userPresent", userPresent)
		c.Next()
	}
}
