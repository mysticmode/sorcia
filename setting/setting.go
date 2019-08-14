package setting

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
)

var conf BaseStruct

// BaseStruct struct
type BaseStruct struct {
	AppMode  string
	Paths    PathsStruct
	Server   ServerStruct
	Postgres PostgresStruct
}

// PathsStruct struct
type PathsStruct struct {
	AssetPath    string
	TemplatePath string
	DataPath     string
	ProjectRoot  string
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
		cfg, err = ini.Load("/home/git/sorcia-core/config/app.ini")
		if err != nil {
			fmt.Printf("Fail to read file: %v", err)
			os.Exit(1)
		}
	}

	conf = BaseStruct{
		AppMode: cfg.Section("").Key("app_mode").String(),
		Paths: PathsStruct{
			AssetPath:    cfg.Section("paths").Key("asset_path").String(),
			TemplatePath: cfg.Section("paths").Key("template_path").String(),
			DataPath:     cfg.Section("paths").Key("data_path").String(),
			ProjectRoot:  cfg.Section("paths").Key("project_root").String(),
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
	}
}

// GetConf ...
func GetConf() *BaseStruct {
	return &conf
}
