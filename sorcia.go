package main

import (
	"fmt"
	"os"

	"sorcia/cmd"
	"sorcia/pkg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'web' / 'usermod' / 'version' subcommands.")
		os.Exit(1)
	}

	// Get config values
	conf := pkg.GetConf()

	switch os.Args[1] {
	case "web":
		cmd.RunWeb(conf)
	case "usermod":
		cmd.UserMod(conf)
	case "version":
		fmt.Println(conf.Version)
	default:
		fmt.Println("Expected 'web' / 'usermod' / 'version' subcommands.")
		os.Exit(1)
	}
}
