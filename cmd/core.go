package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"

	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	// Postgresql driver
	_ "github.com/lib/pq"
	"github.com/urfave/cli"
)

// Core ...
var Core = cli.Command{
	Name:        "core",
	Usage:       "Start web server",
	Description: `This starts the core sorica web server`,
	Action:      runWeb,
}

func runWeb(c *cli.Context) error {
	// Gin initiate
	r := mux.NewRouter()

	// Get config values
	conf := setting.GetConf()

	// Create repositories directory
	// 0755 - The owner can read, write, execute. Everyone else can read and execute but not modify the file.
	os.MkdirAll(path.Join(conf.Paths.DataPath, "repositories"), 0755)

	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	model.CreateAccount(db)
	model.CreateRepo(db)

	// r.Use(
	// 	middleware.CORSMiddleware(),
	// 	middleware.APIMiddleware(db),
	// 	middleware.UserMiddleware(db),
	// )

	r.Use(middleware.Middleware)

	// Gin handlers
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db)
	}).Methods("GET")
	// r.GET("/login", handler.GetLogin)
	// r.POST("/login", handler.PostLogin)
	// r.GET("/logout", handler.GetLogout)
	// r.POST("/register", handler.PostRegister)
	// r.GET("/create", handler.GetCreateRepo)
	// r.POST("/create", handler.PostCreateRepo)
	// r.GET("/+:username", GetHome)
	// r.GET("/+:username/:reponame", handler.GetRepo)
	// r.GET("/+:username/:reponame/tree", handler.GetRepoTree)

	// // Git http backend service handlers
	// r.POST("/+:username/:reponame/git-:rpc", handler.PostServiceRPC)
	// r.GET("/+:username/:reponame/info/refs", handler.GetInfoRefs)
	// r.GET("/+:username/:reponame/HEAD", handler.GetHEADFile)
	// r.GET("/+:username/:reponame/objects/:regex1/:regex2", handler.GetGitRegexRequestHandler)

	staticFileDirectory := http.Dir(conf.Paths.AssetPath)
	// Declare the handler, that routes requests to their respective filename.
	// The fileserver is wrapped in the `stripPrefix` method, because we want to
	// remove the "/public/" prefix when looking for files.
	// For example, if we type "/public/index.html" in our browser, the file server
	// will look for only "index.html" inside the directory declared above.
	// If we did not strip the prefix, the file server would look for
	// "./public/public/index.html", and yield an error
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(staticFileDirectory))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	r.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	http.Handle("/", r)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"POST", "GET"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(r)))

	return nil
}

// IndexPageResponse struct
type IndexPageResponse struct {
	Username string
	repos    *model.GetReposFromUserIDResponse
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := r.Header.Get("sorcia-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		tmpl := template.Must(template.ParseFiles("./templates/index.html"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		data := IndexPageResponse{
			Username: username,
			repos:    repos,
		}

		tmpl.Execute(w, data)
	} else {
		// http.Redirect(w, r, "/login", http.StatusMovedPermanently)
		fmt.Println("token not present")
	}

	// Set cookie example
	// expiration := time.Now().Add(365 * 24 * time.Hour)
	// c := &http.Cookie{Name: "sorcia-token", Value: "abcd", Path: "/", Domain: strings.Split(r.Host, ":")[0], Expires: expiration}
	// http.SetCookie(w, c)
}
