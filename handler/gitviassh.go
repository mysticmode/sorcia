package handler

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var authorizedKey []byte
var gitRPC, gitRepo string

func RunSSH(conf *setting.BaseStruct, db *sql.DB) {
	ssh.Handle(func(s ssh.Session) {
		authorizedKey = gossh.MarshalAuthorizedKey(s.PublicKey())

		if len(s.Command()) == 2 {
			gitRPC = s.Command()[0]
			gitRepo = s.Command()[1]
			fmt.Println(gitRPC)
			fmt.Println(gitRepo)
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

		return
		// io.WriteString(s, fmt.Sprintf("public key used by %s:\n", s.User()))
		// s.Write(authorizedKey)
	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		authKeys := model.GetSSHAllAuthKeys(db)
		fmt.Println(authKeys)
		for _, authKey := range authKeys {
			authKeyByte := []byte(authKey)
			out, _, _, _, err := gossh.ParseAuthorizedKey(authKeyByte)
			errorhandler.CheckError(err)

			isEqual := ssh.KeysEqual(key, out)
			if isEqual {
				fmt.Println("Key matched")
				return true
			}
		}
		// return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
		fmt.Println("Failed to handshake")
		return false
	})

	log.Println("starting ssh server on port 2222...")
	log.Fatal(ssh.ListenAndServe(":22", nil, publicKeyOption, ssh.HostKeyFile(filepath.Join(conf.Paths.SSHPath, "id_rsa"))))
}
