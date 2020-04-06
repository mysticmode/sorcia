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
	m.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		internal.GetMeta(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/password", func(w http.ResponseWriter, r *http.Request) {
		internal.MetaPostPassword(w, r, db, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/site", func(w http.ResponseWriter, r *http.Request) {
		internal.MetaPostSiteSettings(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		internal.GetMetaKeys(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/keys/delete/{keyID}", func(w http.ResponseWriter, r *http.Request) {
		internal.DeleteMetaKey(w, r, db)
	}).Methods("GET")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		internal.PostAuthKey(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		internal.GetMetaUsers(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		internal.PostUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/user/revoke-access/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.RevokeCreateRepoAccess(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/user/add-access/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.AddCreateRepoAccess(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepo(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/meta", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoMeta(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/meta", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoMeta(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/meta/user", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoMetaUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/meta/user/remove/{username}", func(w http.ResponseWriter, r *http.Request) {
		internal.RemoveRepoMetaUser(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/meta/delete", func(w http.ResponseWriter, r *http.Request) {
		internal.PostRepoMetaDelete(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/tree/{branch}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoTree(w, r, db, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/tree/{branchorhash}/{path:[[\\d\\w-_\\.]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoTreePath(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/log/{branch}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoLog(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/commit/{branch}/{hash}", func(w http.ResponseWriter, r *http.Request) {
		internal.GetCommitDetail(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/refs", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoRefs(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/contributors", func(w http.ResponseWriter, r *http.Request) {
		internal.GetRepoContributors(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/dl/{file}", func(w http.ResponseWriter, r *http.Request) {
		internal.ServeRefFile(w, r, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internal.GitviaHTTP(w, r, db, conf)
	}).Methods("GET", "POST")

	staticDir := filepath.Join(conf.Paths.ProjectRoot, "public")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(http.Dir(staticDir)))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	m.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	uploadFileHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(conf.Paths.UploadAssetPath)))
	m.PathPrefix("/uploads/").Handler(uploadFileHandler).Methods("GET")

	return m
}
