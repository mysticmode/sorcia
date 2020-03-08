package util

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	errorhandler "sorcia/error"
)

// PullFromAllBranches ...
func PullFromAllBranches(gitDirPath string) {
	var files []string

	gitPath := GetGitBinPath()

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
		cmd := exec.Command(gitPath, "pull", "origin", fmt.Sprintf("%s:%s", branch, branch))
		cmd.Dir = workDir

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}
	}
}

// GenerateRefs ...
func GenerateRefs(gitDirPath, refsPath, reponame string) {
	var files []string

	gitPath := GetGitBinPath()

	dirSplit := strings.Split(gitDirPath, ".git")
	workDir := dirSplit[0]
	refPath := filepath.Join(gitDirPath, "refs", "tags")
	err := filepath.Walk(refPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	errorhandler.CheckError(err)

	refsPath, err = filepath.Abs(refsPath)
	if err != nil {
		log.Printf("%v", err)
	}

	for i := 0; i < len(files); i++ {
		ss := strings.Split(files[i], "/")
		tag := ss[len(ss)-1]

		if tag == "tags" {
			continue
		}

		tagname := tag
		// Remove 'v' prefix from version
		if strings.HasPrefix(tag, "v") {
			tagname = strings.Split(tag, "v")[1]
		}

		// Generate tar.gz file
		tarFilename := fmt.Sprintf("%s-%s.tar.gz", reponame, tagname)
		tarRefPath := filepath.Join(refsPath, tarFilename)

		if _, err := os.Stat(tarRefPath); os.IsNotExist(err) {
			cmd := exec.Command(gitPath, "archive", "--format=tar.gz", "-o", tarRefPath, tag)
			cmd.Dir = workDir

			var out, stderr bytes.Buffer
			cmd.Stderr = &stderr
			cmd.Stdout = &out

			err := cmd.Run()
			if err != nil {
				log.Printf("%v", err)
			}
		}

		// Generate zip file
		zipFilename := fmt.Sprintf("%s-%s.zip", reponame, tagname)
		zipRefPath := filepath.Join(refsPath, zipFilename)

		if _, err := os.Stat(zipRefPath); os.IsNotExist(err) {
			cmd := exec.Command(gitPath, "archive", "--format=zip", "-o", zipRefPath, tag)
			cmd.Dir = workDir

			var out, stderr bytes.Buffer
			cmd.Stderr = &stderr
			cmd.Stdout = &out

			err = cmd.Run()
			if err != nil {
				log.Printf("%v", err)
			}
		}
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

// CreateRepoDir ...
func CreateRepoDir(repoPath string) {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		err := os.Mkdir(repoPath, os.ModePerm)
		errorhandler.CheckError(err)
	}
}

// CreateRefsDir ...
func CreateRefsDir(refPath string) {
	if _, err := os.Stat(refPath); os.IsNotExist(err) {
		err := os.Mkdir(refPath, os.ModePerm)
		errorhandler.CheckError(err)
	}
}

// GetCommitCounts ...
func GetCommitCounts(repoPath, reponame string) string {
	dirPath := filepath.Join(repoPath, reponame+".git")
	gitPath := GetGitBinPath()
	cmd := exec.Command(gitPath, "rev-list", "HEAD", "--count")
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
	}

	return strings.TrimSpace(out.String())
}
