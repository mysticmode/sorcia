package cmd

import (
	"fmt"
	"log"
	"net/http"

	"sorcia/internal"
	"sorcia/models"
	"sorcia/pkg"
	"sorcia/routes"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// RunWeb ...
func RunWeb(conf *pkg.BaseStruct) {
	// Create necessary directories
	pkg.CreateDir(conf.Paths.RepoPath)
	pkg.CreateDir(conf.Paths.RefsPath)
	pkg.CreateDir(conf.Paths.UploadAssetPath)
	pkg.CreateSSHDirAndGenerateKey(conf.Paths.SSHPath)

	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	models.CreateAccount(db)
	models.CreateSiteSettings(db)
	models.CreateSSHPubKey(db)
	models.CreateRepo(db)
	models.CreateRepoMembers(db)

	go internal.RunSSH(conf, db)

	// Mux initiate
	m := mux.NewRouter()
	m = routes.Router(m, db, conf)

	http.Handle("/", m)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"GET", "POST"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(m)))
}
