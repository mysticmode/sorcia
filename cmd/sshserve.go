package cmd

import (
	"os/exec"

	errorhandler "sorcia/error"

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
	sshcmd := exec.Command("git-shell", "-c", "$SSH_ORIGINAL_COMMAND")
	err := sshcmd.Run()
	errorhandler.CheckError(err)

	return nil
}
