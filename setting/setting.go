package setting

import (
	"database/sql"
	"fmt"
	"os"

	errorhandler "sorcia/error"

	"gopkg.in/ini.v1"
	// PostgreSQL driver
	_ "github.com/lib/pq"
)

var conf BaseStruct

// BaseStruct struct
type BaseStruct struct {
	AppMode  string
	Version  string
	Paths    PathsStruct
	Server   ServerStruct
	Postgres PostgresStruct
	DBConn   *sql.DB
}

// PathsStruct struct
type PathsStruct struct {
	ProjectRoot string
	RepoPath    string
}

// ServerStruct struct
type ServerStruct struct {
	HTTPPort string
}

// PostgresStruct struct
type PostgresStruct struct {
	Hostname string
	Port     string
	Name     string
	Username string
	Password string
	SSLMode  string
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
			ProjectRoot: cfg.Section("paths").Key("project_root").String(),
			RepoPath:    cfg.Section("paths").Key("repo_path").String(),
		},
		Server: ServerStruct{
			HTTPPort: cfg.Section("server").Key("http_port").String(),
		},
		Postgres: PostgresStruct{
			Hostname: cfg.Section("postgres").Key("hostname").String(),
			Port:     cfg.Section("postgres").Key("port").String(),
			Name:     cfg.Section("postgres").Key("name").String(),
			Username: cfg.Section("postgres").Key("username").String(),
			Password: cfg.Section("postgres").Key("password").String(),
			SSLMode:  cfg.Section("postgres").Key("sslmode").String(),
		},
		DBConn: nil,
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", conf.Postgres.Username, conf.Postgres.Password, conf.Postgres.Hostname, conf.Postgres.Port, conf.Postgres.Name, conf.Postgres.SSLMode)
	db, err := sql.Open("postgres", connStr)
	errorhandler.CheckError(err)

	conf.DBConn = db
}

// GetConf ...
func GetConf() *BaseStruct {
	return &conf
}
