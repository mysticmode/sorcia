package cmd

import (
	"fmt"
	"os"
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
	cmdd := exec.Command("git-shell", "-c", os.Getenv("SSH_ORIGINAL_COMMAND"))
	cmdd.Stdout = os.Stdout
	cmdd.Stdin = os.Stdin
	cmdd.Stderr = os.Stderr
	if err := cmdd.Run(); err != nil {
		fmt.Printf("Fail to execute git command: %v", err)
	}

	return nil
}
