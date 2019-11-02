package middleware

import (
	"net/http"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// Adapter type
type Adapter func(http.Handler) http.Handler

// Adapt ...
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

// // APIMiddleware ...
// func APIMiddleware(db *sql.DB) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		c.Set("db", db)
// 		c.Next()
// 	}
// }

// // UserMiddleware ...
// func UserMiddleware(db *sql.DB) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		db, ok := c.MustGet("db").(*sql.DB)
// 		if !ok {
// 			fmt.Println("Middleware db error")
// 		}

// 		token, _ := c.Cookie("sorcia-token")

// 		userID := model.GetUserIDFromToken(db, token)

// 		userPresent := false
// 		if userID != 0 {
// 			userPresent = true
// 		}

// 		c.Set("userPresent", userPresent)
// 		c.Next()
// 	}
// }
