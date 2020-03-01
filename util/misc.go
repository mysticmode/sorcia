package util

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
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

// SSHFingerPrintAndEmail ...
func SSHFingerPrintAndEmail(sshPath string) (string, string) {
	key, err := ioutil.ReadFile(sshPath)
	if err != nil {
		log.Fatal(err)
	}

	parts := strings.Fields(string(key))
	if len(parts) < 2 {
		log.Fatal("bad key")
	}

	var email string
	if parts[2] != "" {
		email = parts[2]
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

	return fingerPrint, email
}
