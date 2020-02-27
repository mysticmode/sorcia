package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	errorhandler "sorcia/error"
)

// PullFromAllBranches ...
func PullFromAllBranches(gitDirPath string) {
	var files []string

	dirSplit := strings.Split(gitDirPath, ".git")
	workDir := dirSplit[0]
	refPath := filepath.Join(gitDirPath, "refs", "heads")
	err := filepath.Walk(refPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	errorhandler.CheckError(err)

	for i := 0; i < len(files); i++ {
		ss := strings.Split(files[i], "/")
		branch := ss[len(ss)-1]

		if branch == "heads" {
			continue
		}

		// by default fast-forward is allowed. Add + to allow non-fast-forward
		cmd := exec.Command("git", "pull", "origin", fmt.Sprintf("+%s:%s", branch, branch))
		cmd.Dir = workDir

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}
		fmt.Println(out.String())
	}
}

// GetGitBinPath ...
func GetGitBinPath() string {
	gitPath := "/usr/bin/git"
	if _, err := os.Stat(gitPath); err != nil {
		gitPath = "/bin/git"
		if _, err = os.Stat(gitPath); err != nil {
			gitPath = "/usr/local/bin/git"
			if _, err = os.Stat(gitPath); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	return gitPath
}
