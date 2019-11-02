package middleware

import (
	"database/sql"
	"fmt"
	"net/http"

	"sorcia/setting"

	_ "github.com/lib/pq"
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
	// cookie, err := r.Cookie("sorcia-token")
	// errorhandler.CheckError(err)

	// fmt.Println("\n\n\n")
	// fmt.Println(cookie)

	// token, err := url.QueryUnescape(cookie.Value)
	// errorhandler.CheckError(err)

	// fmt.Println("\n\n\n")
	// fmt.Println(token)

	// userID := model.GetUserIDFromToken(db, token)

	// userPresent := "false"
	// if userID != 0 {
	// 	userPresent = "true"
	// }

	for _, cookie := range r.Cookies() {
		fmt.Println("Found a cookie named:", cookie.Name)
	}

	w.Header().Set("user-present", "ttt")

	return w
}
