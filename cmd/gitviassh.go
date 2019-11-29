package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sorcia/setting"
	"strings"
)

func parseSSHCmd(cmd string) (string, string) {
	ss := strings.SplitN(cmd, " ", 2)
	if len(ss) != 2 {
		return "", ""
	}

	return ss[0], ss[1]
}

func RunSSH(conf *setting.BaseStruct) {

	sshCmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	if len(sshCmd) == 0 {
		println("Hi there, You've successfully authenticated, but sorcia does not provide shell access.")
		println("If this is unexpected, please log in with password and setup sorcia under another user.")
	}

	gitVerb, args := parseSSHCmd(sshCmd)
	fmt.Println(gitVerb)

	// For Windows
	// gitVerb = strings.Replace(gitVerb, "-", " ", 1)
	// fmt.Println(gitVerb)

	fmt.Println(args)
	fmt.Println(sshCmd)
	repoFullName := strings.ToLower(strings.Trim(args, "'"))
	repoFields := strings.SplitN(repoFullName, "/", 2)
	if len(repoFields) != 2 {
		fmt.Printf("Invalid repository path: %v", args)
	}
	ownerName := strings.ToLower(repoFields[0])
	repoName := strings.TrimSuffix(strings.ToLower(repoFields[1]), ".git")
	repoName = strings.TrimSuffix(repoName, ".wiki")

	fmt.Println(repoFullName)
	fmt.Println(ownerName)
	fmt.Println(repoName)

	if gitVerb != "git-upload-pack" && gitVerb != "git-upload-archive" && gitVerb != "git-receive-pack" {
		fmt.Println("Unknown git command")
	}

	var cmdSSHServe *exec.Cmd
	cmdSSHServe = exec.Command("git-shell", "-c", sshCmd)
	fmt.Println(conf.Paths.RepoPath)
	// cmdSSHServe.Dir = conf.Paths.RepoPath // This should be repo root path
	// cmdSSHServe.Stdout = os.Stdout
	// cmdSSHServe.Stdin = os.Stdin
	// cmdSSHServe.Stderr = os.Stderr
	if err := cmdSSHServe.Run(); err != nil {
		fmt.Printf("Fail to execute git command: %v", err)
	}
}
