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

func GetGitBranches(repoDir string) []string {
	var branches = []string{"master"}

	refPath := filepath.Join(repoDir, "refs", "heads")

	err := filepath.Walk(refPath, func(path string, info os.FileInfo, err error) error {
		branchName := info.Name()
		if branchName != "heads" && branchName != "master" {
			branches = append(branches, branchName)
		}
		return nil
	})
	errorhandler.CheckError("Error on util get git branches filepath walk", err)

	return branches
}

func GetGitTags(repoDir string) ([]string, int) {
	gitPath := GetGitBinPath()

	args := []string{"for-each-ref", "--sort=-taggerdate", "--format", "%(refname)", "refs/tags"}
	out := ForkExec(gitPath, args, repoDir)

	lines := strings.Fields(out)

	var tags []string

	for _, line := range lines {
		tags = append(tags, strings.Split(line, "/")[2])
	}

	return tags, len(tags)
}

// GenerateRefs
func GenerateRefs(refsPath, repoPath, repoGitName string) {
	gitPath := GetGitBinPath()

	repoDir := filepath.Join(repoPath, repoGitName)
	tags, _ := GetGitTags(repoDir)

	// example.git -> example
	repoName := strings.TrimSuffix(repoGitName, ".git")

	for _, tag := range tags {

		tagname := tag

		// Remove 'v' prefix from version
		if strings.HasPrefix(tag, "v") {
			tagname = strings.Split(tag, "v")[1]
		}

		// Generate tar.gz file
		tarFilename := fmt.Sprintf("%s-%s.tar.gz", repoName, tagname)
		tarRefPath := filepath.Join(refsPath, tarFilename)

		if _, err := os.Stat(tarRefPath); os.IsNotExist(err) {
			args := []string{"archive", "--format=tar.gz", "-o", tarRefPath, tag}
			_ = ForkExec(gitPath, args, repoDir)
		}

		// Generate zip file
		zipFilename := fmt.Sprintf("%s-%s.zip", repoName, tagname)
		zipRefPath := filepath.Join(refsPath, zipFilename)

		if _, err := os.Stat(zipRefPath); os.IsNotExist(err) {
			args := []string{"archive", "--format=zip", "-o", zipRefPath, tag}
			_ = ForkExec(gitPath, args, repoDir)
		}
	}
}

// UpdateRefsWithNewName ...
func UpdateRefsWithNewName(refsPath, repoPath, oldRepoGitName, newRepoGitName string) {
	refsPattern := filepath.Join(refsPath, oldRepoGitName+"*")

	files, err := filepath.Glob(refsPattern)
	errorhandler.CheckError("Error on updaterefspath filepath.Glob", err)

	for _, f := range files {
		err := os.Remove(f)
		errorhandler.CheckError("Error on removing ref files", err)
	}

	go GenerateRefs(refsPath, repoPath, newRepoGitName+".git")
}

// GetCommitCounts ...
func GetCommitCounts(repoPath, reponame string) string {
	dirPath := filepath.Join(repoPath, reponame+".git")
	gitPath := GetGitBinPath()

	args := []string{"rev-list", "--count", "HEAD"}
	out := ForkExec(gitPath, args, dirPath)

	return strings.TrimSpace(out)
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

// ForkExec ...
func ForkExec(legendArg string, restArgs []string, dirPath string) string {
	cmd := exec.Command(legendArg, restArgs...)
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Printf(stderr.String())
	}

	return out.String()
}
