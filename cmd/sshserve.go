package cmd

import (
	"bytes"
	"fmt"
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
	cc := exec.Command("git-shell", "-c", "'$SSH_ORIGINAL_COMMAND'")

	var out bytes.Buffer
	cc.Stdout = &out

	err := cc.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("in all caps: %q\n", out.String())

	return nil
}
