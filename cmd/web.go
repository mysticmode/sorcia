package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"
	"sorcia/util"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// RunWeb ...
func RunWeb(conf *setting.BaseStruct) {
	// Create necessary directories
	util.CreateDir(conf.Paths.RepoPath)
	util.CreateDir(conf.Paths.RefsPath)
	util.CreateDir(conf.Paths.UploadAssetPath)
	util.CreateSSHDirAndGenerateKey(conf.Paths.SSHPath)

	// Mux initiate
	m := mux.NewRouter()

	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	model.CreateAccount(db)
	model.CreateSiteSettings(db)
	model.CreateSSHPubKey(db)
	model.CreateRepo(db)

	go handler.RunSSH(conf, db)

	m.Use(middleware.Middleware)

	// Web handlers
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogin(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.PostLogin(w, r, db, conf.Version, decoder)
	}).Methods("POST")
	m.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogout(w, r)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCreateRepo(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.PostCreateRepo(w, r, db, decoder, conf.Version, conf.Paths.RepoPath)
	}).Methods("POST")
	m.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMeta(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/password", func(w http.ResponseWriter, r *http.Request) {
		handler.MetaPostPassword(w, r, db, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/site", func(w http.ResponseWriter, r *http.Request) {
		handler.MetaPostSiteSettings(w, r, db, conf.Paths.UploadAssetPath)
	}).Methods("POST")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMetaKeys(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		handler.PostAuthKey(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMetaUsers(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepo(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/tree/{branch}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTree(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/tree/{branch}/{path:[[\\d\\w-_\\.]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTreePath(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/log/{branch}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoLog(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/refs", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoRefs(w, r, db, conf.Version, conf.Paths.RepoPath, conf.Paths.RefsPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/contributors", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoContributors(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/dl/{file}", func(w http.ResponseWriter, r *http.Request) {
		handler.ServeRefFile(w, r, conf.Paths.RefsPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GitviaHTTP(w, r, db, conf.Paths.RepoPath, conf.Paths.RepoPath, conf.Paths.RefsPath)
	}).Methods("GET", "POST")

	staticDir := filepath.Join(conf.Paths.ProjectRoot, "public")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(http.Dir(staticDir)))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	m.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	uploadFileHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(conf.Paths.UploadAssetPath)))
	m.PathPrefix("/uploads/").Handler(uploadFileHandler).Methods("GET")

	http.Handle("/", m)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"GET", "POST"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(m)))
}

// IndexPageResponse struct
type IndexPageResponse struct {
	IsLoggedIn       bool
	ShowLoginMenu    bool
	HeaderActiveMenu string
	SorciaVersion    string
	Repos            *model.GetReposFromUserIDResponse
	AllPublicRepos   *model.GetAllPublicReposResponse
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		indexPage := path.Join("./templates", "index.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "",
			SorciaVersion:    sorciaVersion,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		if !model.CheckIfFirstUserExists(db) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		indexPage := path.Join("./templates", "index.html")
		footerPage := path.Join("./templates", "footer.html")
		repos := model.GetAllPublicRepos(db)

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsLoggedIn:     false,
			ShowLoginMenu:  true,
			SorciaVersion:  sorciaVersion,
			AllPublicRepos: repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	}
}
