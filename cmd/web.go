package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/urfave/cli"

	// PostgreSQL driver
	_ "github.com/lib/pq"
)

// Web ...
var Web = cli.Command{
	Name:        "web",
	Usage:       "Start web server",
	Description: `This starts the sorica web server`,
	Action:      runWeb,
}

var decoder = schema.NewDecoder()

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

	// Web handlers
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db, conf.Paths.TemplatePath)
	}).Methods("GET")
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogin(w, r, db, conf.Paths.TemplatePath)
	}).Methods("GET")
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.PostLogin(w, r, db, conf.Paths.DataPath, conf.Paths.TemplatePath, decoder)
	}).Methods("POST")
	r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogout(w, r)
	}).Methods("GET")
	r.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCreateRepo(w, r, db, conf.Paths.TemplatePath)
	}).Methods("GET")
	r.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.PostCreateRepo(w, r, db, conf.Paths.DataPath, decoder)
	}).Methods("POST")
	r.HandleFunc("/+{username}/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepo(w, r, db, conf.Paths.TemplatePath)
	}).Methods("GET")
	r.HandleFunc("/+{username}/{reponame}/tree", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTree(w, r, db, conf.Paths.TemplatePath)
	}).Methods("GET")

	// Git http backend service handlers
	r.HandleFunc("/+:{username}/{reponame}/git-{rpc}", func(w http.ResponseWriter, r *http.Request) {
		handler.PostServiceRPC(w, r, db, conf.Paths.RepoPath)
	}).Methods("POST")
	r.HandleFunc("/+:{username}/{reponame}/info/refs", func(w http.ResponseWriter, r *http.Request) {
		handler.GetInfoRefs(w, r, db, conf.Paths.RepoPath)
	}).Methods("GET")
	r.HandleFunc("/+:{username}/{reponame}/HEAD", func(w http.ResponseWriter, r *http.Request) {
		handler.GetHEADFile(w, r, db, conf.Paths.RepoPath)
	}).Methods("GET")
	r.HandleFunc("/+:{username}/{reponame}/objects/{regex1}/{regex2}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetGitRegexRequestHandler(w, r, db, conf.Paths.RepoPath)
	}).Methods("GET")

	staticFileDirectory := http.Dir(conf.Paths.AssetPath)
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(staticFileDirectory))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	r.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	http.Handle("/", r)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"GET", "POST"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(r)))

	return nil
}

// IndexPageResponse struct
type IndexPageResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	Username         string
	Repos            *model.GetReposFromUserIDResponse
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, templatePath string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		lp := path.Join(templatePath, "templates", "layout.tmpl")
		hp := path.Join(templatePath, "templates", "header.tmpl")
		ip := path.Join(templatePath, "templates", "index.tmpl")

		tmpl, err := template.ParseFiles(lp, hp, ip)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "header__menu--dashboard",
			Username:         username,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
