package pkg

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		log.Printf("bad key")
	}

	k, err := base64.StdEncoding.DecodeString(parts[1])
	CheckError("Error on util ssh fingerprint decode string", err)

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

// CreateDir ...
func CreateDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		CheckError("Error on util create dir", err)
	}
}

// CreateSSHDirAndGenerateKey ...
func CreateSSHDirAndGenerateKey(sshPath string) {
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		err := os.MkdirAll(sshPath, os.ModePerm)
		CheckError("Error on util create ssh dir and generate ssh key", err)
	}

	keyPath := filepath.Join(sshPath, "id_rsa")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		args := []string{"-f", keyPath, "-t", "rsa", "-m", "PEM", "-N", ""}
		_ = ForkExec("ssh-keygen", args, ".")
	}
}

// LimitCharLengthInString ...
func LimitCharLengthInString(limitString string) string {
	if len(limitString) > 50 {
		limitString = fmt.Sprintf("%s...", string(limitString[:50]))
		return limitString
	}

	return limitString
}

// ContainsValueInArr tells whether a contains x.
func ContainsValueInArr(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
