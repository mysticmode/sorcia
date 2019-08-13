package cmd

import (
	"log"
	"os/exec"

	"github.com/urfave/cli"
)

// SSHServe ...
var SSHServe = cli.Command{
	Name:        "sshserve",
	Usage:       "Start web server",
	Description: `This serves git via SSH`,
	Action:      runSSH,
}

func runSSH(c *cli.Context) error {
	sshcmd := exec.Command("/bin/sh", "-c", "./gitserve")
	err := sshcmd.Run()
	log.Printf("Command finished with error: %v", err)

	return nil
}
