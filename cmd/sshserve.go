package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sorcia/setting"

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

	// Get config values
	conf := setting.GetConf()

	sshCmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	if len(sshCmd) == 0 {
		println("Hi there, You've successfully authenticated, but sorcia does not provide shell access.")
		println("If this is unexpected, please log in with password and setup sorcia under another user.")
		return nil
	}

	cmdd := exec.Command("git-shell", "-c", os.Getenv("SSH_ORIGINAL_COMMAND"))
	cmdd.Dir = conf.Paths.DataPath // This should be repo root path
	cmdd.Stdout = os.Stdout
	cmdd.Stdin = os.Stdin
	cmdd.Stderr = os.Stderr
	if err := cmdd.Run(); err != nil {
		fmt.Printf("Fail to execute git command: %v", err)
	}

	return nil
}
