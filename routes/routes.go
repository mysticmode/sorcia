package routes

import (
	"database/sql"
	"net/http"
	"path/filepath"

	"sorcia/internal"
	"sorcia/middleware"
	"sorcia/pkg"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// Router ...
func Router(m *mux.Router, db *sql.DB, conf *pkg.BaseStruct) *mux.Router {

	m.Use(middleware.Middleware)

	// Web handlers
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		internal.GetHome(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		internal.GetLogin(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		internal.PostLogin(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		internal.GetLogout(w, r)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		internal.GetCreateRepo(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		internal.PostCreateRepo(w, r, db, decoder, conf)
	}).Methods("POST")
	m.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		internal.GetSettings(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/settings/password", func(w http.ResponseWriter, r *http.Request) {
		internal.SettingsPostPassword(w, r, db, decoder)
	}).Methods("POST")
	m.HandleFunc("/settings/site", func(w http.ResponseWriter, r *http.Request) {
		internal.SettingsPostSiteSettings(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/settings/keys", func(w http.ResponseWriter, r *http.Request) {
		internal.GetSettingsKeys(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/settings/keys/delete/{keyID}", func(w http.ResponseWriter, r *http.Request) {
		internal.DeleteSettingsKey(w, r, db)
	}).Methods("GET")
	m.HandleFunc("/settings/keys", func(w http.ResponseWriter, r *http.Request) {
		internal.PostAuthKey(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/settings/users", func(w http.ResponseWriter, r *http.Request) {
		internal.GetSettingsUsers(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/settings/users", func(w http.ResponseWriter, r *http.Request) {
		internal.PostUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/settings/user/revoke-access/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.RevokeCreateRepoAccess(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/settings/user/add-access/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.AddCreateRepoAccess(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepo(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/settings", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoSettings(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/settings", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoSettings(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/settings/user", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoSettingsUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/settings/user/remove/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.RemoveRepoSettingsUser(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/settings/delete", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoSettingsDelete(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/browse/{branch}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoBrowse(w, r, db, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/browse/{branchorhash}/{path:[[\\d\\w-_\\.]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoBrowsePath(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/commits/{branch}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoCommits(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/commit/{branch}/{hash}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetCommitDetail(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/releases", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoRefs(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/contributors", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoContributors(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/dl/{file}", func(w http.ResponseWriter, r *http.Request) {
		internal.ServeReleasesFile(w, r, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GitviaHTTP(w, r, db, conf)
	}).Methods("GET", "POST")

	staticDir, err := filepath.Abs(filepath.Join(conf.Paths.ProjectRoot, "public"))
	pkg.CheckError("static absolute path failed", err)

	staticFileHandler := http.StripPrefix("/public/", http.FileServer(http.Dir(staticDir)))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	m.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	uploadFileHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(conf.Paths.UploadAssetPath)))
	m.PathPrefix("/uploads/").Handler(uploadFileHandler).Methods("GET")

	return m
}
