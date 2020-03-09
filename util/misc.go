package util

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	errorhandler "sorcia/error"
)

// IsAlnumOrHyphen ...
func IsAlnumOrHyphen(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// SSHFingerPrint ...
func SSHFingerPrint(authKey string) string {
	parts := strings.Fields(string(authKey))
	if len(parts) < 2 {
		log.Fatal("bad key")
	}

	k, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		log.Fatal(err)
	}

	fp := md5.Sum([]byte(k))
	var fingerPrint string
	for i, b := range fp {
		fingerPrint = fmt.Sprintf("%s%02x", fingerPrint, b)
		if i < len(fp)-1 {
			fingerPrint = fmt.Sprintf("%s:", fingerPrint)
		}
	}

	return fingerPrint
}

// CreateSSHDirAndGenerateKey ...
func CreateSSHDirAndGenerateKey(sshPath string) {
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		err := os.Mkdir(sshPath, os.ModePerm)
		errorhandler.CheckError(err)
	}

	keyPath := filepath.Join(sshPath, "id_rsa")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		args := []string{"-f", keyPath, "-t", "rsa", "-m", "PEM", "-N", ""}
		_ = ForkExec("ssh-keygen", args, ".")
	}
}

func LimitCharLengthInString(limitString string) string {
	strManipulate := fmt.Sprintf("%s...", string(limitString[:50]))
	return strManipulate
}
