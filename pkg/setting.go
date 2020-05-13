package pkg

import (
	"database/sql"
	"fmt"
	"os"
	"path"

	// SQLite3 driver
	_ "github.com/mattn/go-sqlite3"

	"gopkg.in/ini.v1"
)

var conf BaseStruct

// BaseStruct struct
type BaseStruct struct {
	AppMode string
	Version string
	Paths   PathsStruct
	Server  ServerStruct
	DBConn  *sql.DB
}

// PathsStruct struct
type PathsStruct struct {
	ProjectRoot     string
	RepoPath        string
	RefsPath        string
	DBPath          string
	SSHPath         string
	TemplatePath    string
	UploadAssetPath string
}

// ServerStruct struct
type ServerStruct struct {
	HTTPPort string
	SSHPort  string
}

func init() {
	cfg, err := ini.Load("config/app.ini")
	if err != nil {
		cfg, err = ini.Load("/home/git/sorcia/config/app.ini")
		if err != nil {
			fmt.Printf("Fail to read file: %v", err)
			os.Exit(1)
		}
	}

	conf = BaseStruct{
		AppMode: cfg.Section("").Key("app_mode").String(),
		Version: cfg.Section("").Key("version").String(),
		Paths: PathsStruct{
			ProjectRoot:     cfg.Section("paths").Key("project_root").String(),
			RepoPath:        cfg.Section("paths").Key("repo_path").String(),
			RefsPath:        cfg.Section("paths").Key("refs_path").String(),
			DBPath:          cfg.Section("paths").Key("db_path").String(),
			SSHPath:         cfg.Section("paths").Key("ssh_path").String(),
			TemplatePath:    cfg.Section("paths").Key("template_path").String(),
			UploadAssetPath: cfg.Section("paths").Key("upload_asset_path").String(),
		},
		Server: ServerStruct{
			HTTPPort: cfg.Section("server").Key("http_port").String(),
			SSHPort:  cfg.Section("server").Key("ssh_port").String(),
		},
		DBConn: nil,
	}

	db, err := sql.Open("sqlite3", path.Join(conf.Paths.DBPath, "sorcia.db?_foreign_keys=on"))
	CheckError("Error on opening sqlite3", err)

	conf.DBConn = db
}

// GetConf ...
func GetConf() *BaseStruct {
	return &conf
}
