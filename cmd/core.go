package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

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
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", conf.Postgres.Username, conf.Postgres.Password, conf.Postgres.Hostname, conf.Postgres.Port, conf.Postgres.Name, conf.Postgres.SSLMode)
	db, err := sql.Open("postgres", connStr)
	errorhandler.CheckError(err)
	defer db.Close()

	model.CreateAccount(db)
	model.CreateRepo(db)

	// r.Use(
	// 	middleware.CORSMiddleware(),
	// 	middleware.APIMiddleware(db),
	// 	middleware.UserMiddleware(db),
	// )

	// Gin handlers
	r.HandleFunc("/", GetHome).Methods("GET")
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

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:1937",
		// Good practice: enforce timeouts for servers you create!
		// WriteTimeout: 15 * time.Second,
		// ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

	// Listen and serve on 1937
	// http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), nil)

	return nil
}

// IndexPageData struct
type IndexPageData struct {
	Username string
	repos    []string
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request) {
	// db, ok := c.MustGet("db").(*sql.DB)
	// if !ok {
	// 	fmt.Println("Middleware db error")
	// }

	// userPresent, ok := c.MustGet("userPresent").(bool)
	// if !ok {
	// 	fmt.Println("Middleware user error")
	// }

	// if userPresent {
	// 	token, _ := c.Cookie("sorcia-token")
	// 	userID := model.GetUserIDFromToken(db, token)
	// 	username := model.GetUsernameFromToken(db, token)

	// 	repos := model.GetReposFromUserID(db, userID)

	// 	c.HTML(200, "index.html", gin.H{
	// 		"username": username,
	// 		"repos":    repos,
	// 	})
	// } else {
	// 	c.Redirect(http.StatusMovedPermanently, "/login")
	// }

	tmpl := template.Must(template.ParseFiles("./templates/index.html"))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := IndexPageData{
		Username: "mysticmode",
		repos:    nil,
	}

	tmpl.Execute(w, data)
}
