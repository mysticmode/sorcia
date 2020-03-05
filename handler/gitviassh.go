package handler

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"
	"sorcia/util"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var authorizedKey []byte
var gitRPC, gitRepo string
var userIDs []string
var reponame string
var repoAccess bool

func RunSSH(conf *setting.BaseStruct, db *sql.DB) {
	ssh.Handle(func(s ssh.Session) {
		authorizedKey = gossh.MarshalAuthorizedKey(s.PublicKey())

		repoAccess = false

		if len(s.Command()) == 2 {
			gitRPC = s.Command()[0]
			gitRepo = s.Command()[1]

			if strings.HasPrefix(gitRepo, "/") {
				gitRepo = strings.Split(gitRepo, "/")[1]
			}

			if !strings.HasSuffix(gitRepo, ".git") {
				log.Printf("ssh: invalid git repository name")
				return
			} else {
				reponame = strings.Split(gitRepo, ".git")[0]
			}

			for _, userID := range userIDs {
				userIDInt, err := strconv.Atoi(userID)
				if err != nil {
					log.Printf("ssh: cannot convert userID to integer")
					return
				}

				if model.CheckRepoAccessFromUserIDAndReponame(db, userIDInt, reponame) {
					repoAccess = true
				}
			}

			if !repoAccess {
				log.Printf("ssh: no repo access")
				return
			}
		} else {
			log.Printf("ssh: no git command")
			return
		}

		cmd := exec.Command(gitRPC, gitRepo)
		cmd.Dir = conf.Paths.RepoPath

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("ssh: cant open stdout pipe: %v", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("ssh: cant open stderr pipe: %v", err)
			return
		}

		input, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("ssh: cant open stdin pipe: %v", err)
			return
		}

		if err = cmd.Start(); err != nil {
			log.Printf("ssh: start error: %v", err)
			return
		}

		go io.Copy(input, s)
		io.Copy(s, stdout)
		io.Copy(s.Stderr(), stderr)

		if err = cmd.Wait(); err != nil {
			log.Printf("ssh: command failed: %v", err)
			return
		}

		s.SendRequest("exit-status", false, []byte{0, 0, 0, 0})

		if gitRPC == "git-receive-pack" {
			repoDir := filepath.Join(conf.Paths.RepoPath, reponame)
			go util.PullFromAllBranches(repoDir)
		}

		return
	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		sshDetail := model.GetSSHAllAuthKeys(db)
		userIDs = sshDetail.UserIDs

		for i := 0; i < len(sshDetail.AuthKeys); i++ {
			authKeyByte := []byte(sshDetail.AuthKeys[i])
			allowed, _, _, _, err := gossh.ParseAuthorizedKey(authKeyByte)
			errorhandler.CheckError(err)

			if ssh.KeysEqual(key, allowed) {
				return true
			}
		}
		log.Printf("Failed to handshake")
		return false
	})

	log.Printf("Starting ssh server on port %s...", conf.Server.SSHPort)
	sshPort := fmt.Sprintf(":%s", conf.Server.SSHPort)
	log.Fatal(ssh.ListenAndServe(sshPort, nil, ssh.NoPty(), publicKeyOption, ssh.HostKeyFile(filepath.Join(conf.Paths.SSHPath, "id_rsa"))))
}
