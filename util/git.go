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
		args := []string{"pull", "origin", fmt.Sprintf("%s:%s", branch, branch)}
		_ = ForkExec(gitPath, args, workDir)
	}
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

// GetCommitCounts ...
func GetCommitCounts(repoPath, reponame string) string {
	dirPath := filepath.Join(repoPath, reponame+".git")
	gitPath := GetGitBinPath()

	args := []string{"rev-list", "HEAD", "--count"}
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

// CreateDir
func CreateDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModePerm)
		errorhandler.CheckError(err)
	}
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
