package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'web' or 'gitviassh' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "web":
		cmd.runWeb()
	case "gitviassh":
		cmd.runSSH()
	default:
		fmt.Println("Expected 'web' or 'gitviassh' subcommands")
		os.Exit(1)
	}
}
