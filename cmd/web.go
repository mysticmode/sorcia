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
	model.CreateRepoMembers(db)

	go handler.RunSSH(conf, db)

	m.Use(middleware.Middleware)

	// Web handlers
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogin(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.PostLogin(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogout(w, r)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCreateRepo(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.PostCreateRepo(w, r, db, decoder, conf)
	}).Methods("POST")
	m.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMeta(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/password", func(w http.ResponseWriter, r *http.Request) {
		handler.MetaPostPassword(w, r, db, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/site", func(w http.ResponseWriter, r *http.Request) {
		handler.MetaPostSiteSettings(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMetaKeys(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/keys/delete/{keyID}", func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteMetaKey(w, r, db)
	}).Methods("GET")
	m.HandleFunc("/meta/keys", func(w http.ResponseWriter, r *http.Request) {
		handler.PostAuthKey(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMetaUsers(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/meta/users", func(w http.ResponseWriter, r *http.Request) {
		handler.PostUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepo(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/meta", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoMeta(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/meta", func(w http.ResponseWriter, r *http.Request) {
		handler.PostRepoMeta(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/meta/user", func(w http.ResponseWriter, r *http.Request) {
		handler.PostRepoMetaUser(w, r, db, conf, decoder)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/meta/delete", func(w http.ResponseWriter, r *http.Request) {
		handler.PostRepoMetaDelete(w, r, db, conf)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}/tree/{branch}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTree(w, r, db, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/tree/{branchorhash}/{path:[[\\d\\w-_\\.]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTreePath(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/log/{branch}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoLog(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/commit/{branch}/{hash}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCommitDetail(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/refs", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoRefs(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/contributors", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoContributors(w, r, db, conf)
	}).Methods("GET")
	m.HandleFunc("/dl/{file}", func(w http.ResponseWriter, r *http.Request) {
		handler.ServeRefFile(w, r, conf)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GitviaHTTP(w, r, db, conf)
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
	CanCreateRepo    bool
	Repos            GetReposStruct
	SiteSettings     util.SiteSettings
}

// GetReposStruct struct
type GetReposStruct struct {
	Repositories []RepoDetailStruct
}

// ReposDetailStruct struct
type RepoDetailStruct struct {
	ID          int
	Name        string
	Description string
	IsPrivate   bool
	Permission  string
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	repos := model.GetAllPublicRepos(db)
	var grs GetReposStruct

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		for _, repo := range repos.Repositories {
			rd := RepoDetailStruct{
				ID:          repo.ID,
				Name:        repo.Name,
				Description: repo.Description,
				IsPrivate:   repo.IsPrivate,
				Permission:  repo.Permission,
			}
			if model.CheckRepoMemberExistFromUserIDAndRepoID(db, userID, repo.ID) {
				rd.Permission = model.GetRepoMemberPermissionFromUserIDAndRepoID(db, userID, repo.ID)
			} else if model.CheckRepoOwnerFromUserIDAndReponame(db, userID, repo.Name) {
				rd.Permission = "read/write"
			}

			grs.Repositories = append(grs.Repositories, rd)
		}

		var reposAsMember model.GetReposStruct
		var repoAsMember model.RepoDetailStruct

		reposAsMember = model.GetReposFromUserID(db, userID)

		repoIDs := model.GetRepoIDsOnRepoMembersUsingUserID(db, userID)
		for _, repoID := range repoIDs {
			repoAsMember = model.GetRepoFromRepoID(db, repoID)
			reposAsMember.Repositories = append(reposAsMember.Repositories, repoAsMember)
		}

		for _, repo := range reposAsMember.Repositories {
			repoExistCount := 0
			for _, publicRepo := range grs.Repositories {
				if publicRepo.Name == repo.Name {
					repoExistCount = 1
				}
			}

			if repoExistCount == 0 {
				rd := RepoDetailStruct{
					ID:          repo.ID,
					Name:        repo.Name,
					Description: repo.Description,
					IsPrivate:   repo.IsPrivate,
					Permission:  repo.Permission,
				}
				if model.CheckRepoMemberExistFromUserIDAndRepoID(db, userID, repo.ID) {
					rd.Permission = model.GetRepoMemberPermissionFromUserIDAndRepoID(db, userID, repo.ID)
				} else if model.CheckRepoOwnerFromUserIDAndReponame(db, userID, repo.Name) {
					rd.Permission = "read/write"
				}

				grs.Repositories = append(grs.Repositories, rd)
			}
		}

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
			SorciaVersion:    conf.Version,
			CanCreateRepo:    model.CheckifUserCanCreateRepo(db, userID),
			Repos:            grs,
			SiteSettings:     util.GetSiteSettings(db, conf),
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

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsLoggedIn:    false,
			ShowLoginMenu: true,
			SorciaVersion: conf.Version,
			Repos:         grs,
			SiteSettings:  util.GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	}
}
