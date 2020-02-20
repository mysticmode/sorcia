package middleware

import (
	"database/sql"
	"net/http"

	"sorcia/model"
	"sorcia/setting"

	// SQLite3 driver
	_ "github.com/mattn/go-sqlite3"
)

var middlewareDB *sql.DB

func init() {
	// Get config values
	conf := setting.GetConf()

	// Open postgres database
	db := conf.DBConn

	middlewareDB = db
}

// Middleware ...
func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = userMiddleware(w, r, middlewareDB)
		h.ServeHTTP(w, r)
	})
}

func userMiddleware(w http.ResponseWriter, r *http.Request, db *sql.DB) http.ResponseWriter {
	cookieName := "sorcia-token"
	var cookieValue string
	userPresent := "false"
	for _, cookie := range r.Cookies() {
		if cookie.Name == cookieName && cookie.Value != "" {
			cookieValue = cookie.Value
			userID := model.GetUserIDFromToken(db, cookie.Value)
			if userID != 0 {
				userPresent = "true"
			}
		}
	}

	w.Header().Set("sorcia-cookie-token", cookieValue)
	w.Header().Set("user-present", userPresent)

	return w
}
