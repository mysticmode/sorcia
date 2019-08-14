package cmd

import (
	"bytes"
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
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmdd.Stdout = &out
	cmdd.Stderr = &stderr
	err := cmdd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return nil
	}
	fmt.Println("Result: " + out.String())
	return nil
}
