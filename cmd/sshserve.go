package cmd

import (
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
	out, err := exec.Command("/bin/bash", "-c", "./gitserve").Output()
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	fmt.Printf("%s\n", out)

	cmd := exec.Command("/bin/bash", "-c", "./gitserve")
	log.Printf("Running command and waiting for it to finish...")
	err = cmd.Run()
	log.Printf("Command finished with error: %v", err)

	return nil
}
