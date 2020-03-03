package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

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
	// Mux initiate
	m := mux.NewRouter()

	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	model.CreateAccount(db)
	model.CreateSSHPubKey(db)
	model.CreateRepo(db)

	var wg sync.WaitGroup
	c := make(chan bool)
	wg.Add(1)
	go func() {
		_, ok := <-c
		if !ok {
			fmt.Println("SSH server is shutting down.")
			defer wg.Done()
		}
		RunSSH(conf)
	}()
	wg.Wait()

	m.Use(middleware.Middleware)

	// Web handlers
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogin(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.PostLogin(w, r, db, conf.Version, decoder, conf.Paths.RepoPath)
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
		GetMeta(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		GetMetaKeys(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		PostAuthKey(w, r, db, conf.Version, conf.Paths.SSHPath, conf, decoder, c)
	}).Methods("POST")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		GetMetaUsers(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepo(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/tree", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTree(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/tree/{[[\\d\\w-_\\.]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTreePath(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/log", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoLog(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GitviaHTTP(w, r, db, conf.Paths.RepoPath)
	}).Methods("GET", "POST")

	staticDir := filepath.Join(conf.Paths.ProjectRoot, "public")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(http.Dir(staticDir)))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	m.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	http.Handle("/", m)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"GET", "POST"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(m)))
}

// IndexPageResponse struct
type IndexPageResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	SorciaVersion    string
	Username         string
	Repos            *model.GetReposFromUserIDResponse
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		indexPage := path.Join("./templates", "index.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "",
			SorciaVersion:    sorciaVersion,
			Username:         username,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// GetMeta ...
func GetMeta(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
			Username:         username,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// MetaKeysResponse struct
type MetaKeysResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	SorciaVersion    string
	SSHKeys          *model.SSHKeysResponse
}

// GetMetaKeys ...
func GetMetaKeys(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		sshKeys := model.GetSSHKeysFromUserId(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta-keys.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := MetaKeysResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
			SSHKeys:          sshKeys,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// CreateAuthKeyRequest struct
type CreateAuthKeyRequest struct {
	Title   string `schema:"sshtitle"`
	AuthKey string `schema:"sshkey"`
}

// PostAuthKey ...
func PostAuthKey(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string, sshPath string, conf *setting.BaseStruct, decoder *schema.Decoder, c chan bool) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &errorhandler.Response{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	var createAuthKeyRequest = &CreateAuthKeyRequest{}
	err := decoder.Decode(createAuthKeyRequest, r.PostForm)
	errorhandler.CheckError(err)

	token := w.Header().Get("sorcia-cookie-token")
	userID := model.GetUserIDFromToken(db, token)

	authKey := strings.TrimSpace(createAuthKeyRequest.AuthKey)
	fingerPrint := util.SSHFingerPrint(authKey)

	ispk := model.InsertSSHPubKeyStruct{
		AuthKey:     authKey,
		Title:       strings.TrimSpace(createAuthKeyRequest.Title),
		Fingerprint: fingerPrint,
		UserID:      userID,
	}

	model.InsertSSHPubKey(db, ispk)

	keyPath := filepath.Join(sshPath, "authorized_keys")
	f, err := os.OpenFile(keyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	errorhandler.CheckError(err)
	defer f.Close()

	if _, err := f.WriteString(authKey + "\n"); err != nil {
		log.Println(err)
	}

	close(c)

	var wg sync.WaitGroup
	c = make(chan bool)
	wg.Add(1)
	go func() {
		_, ok := <-c
		if !ok {
			fmt.Println("SSH server is shutting down.")
			defer wg.Done()
		}
		RunSSH(conf)
	}()
	wg.Wait()

	http.Redirect(w, r, "/meta/keys", http.StatusFound)
}

// GetMetaUsers ...
func GetMetaUsers(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		metaPage := path.Join("./templates", "meta-users.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, metaPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "meta",
			SorciaVersion:    sorciaVersion,
			Username:         username,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
