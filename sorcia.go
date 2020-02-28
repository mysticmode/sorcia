package main

import (
	"fmt"
	"os"

	"sorcia/cmd"
	"sorcia/setting"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'web' or 'gitviassh' or 'version' subcommands")
		os.Exit(1)
	}

	// Get config values
	conf := setting.GetConf()

	switch os.Args[1] {
	case "web":
		cmd.RunWeb(conf)
	case "gitviassh":
		cmd.RunSSH(conf)
	case "usermod":
		cmd.UserMod(conf)
	case "version":
		fmt.Println(conf.Version)
	default:
		fmt.Println("Expected 'web' or 'gitviassh' or 'version' subcommands")
		os.Exit(1)
	}
}
