package util

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"
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
	errorhandler.CheckError("Error on util ssh fingerprint decode string", err)

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

// CreateDir
func CreateDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		errorhandler.CheckError("Error on util create dir", err)
	}
}

// CreateSSHDirAndGenerateKey ...
func CreateSSHDirAndGenerateKey(sshPath string) {
	if _, err := os.Stat(sshPath); os.IsNotExist(err) {
		err := os.MkdirAll(sshPath, os.ModePerm)
		errorhandler.CheckError("Error on util create ssh dir and generate ssh key", err)
	}

	keyPath := filepath.Join(sshPath, "id_rsa")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		args := []string{"-f", keyPath, "-t", "rsa", "-m", "PEM", "-N", ""}
		_ = ForkExec("ssh-keygen", args, ".")
	}
}

func LimitCharLengthInString(limitString string) string {
	if len(limitString) > 50 {
		limitString = fmt.Sprintf("%s...", string(limitString[:50]))
		return limitString
	}

	return limitString
}

// Contains tells whether a contains x.
func ContainsValueInArr(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// SiteSettings struct
type SiteSettings struct {
	IsSiteTitle    bool
	IsSiteFavicon  bool
	IsSiteLogo     bool
	SiteTitle      string
	SiteFavicon    string
	SiteFaviconExt string
	SiteLogo       string
	SiteLogoWidth  string
	SiteLogoHeight string
	IsSiteLogoSVG  bool
	SVGDAT         template.HTML
}

// GetSiteSettings ...
func GetSiteSettings(db *sql.DB, conf *setting.BaseStruct) SiteSettings {
	gssr := model.GetSiteSettings(db, conf)

	isSiteTitle := true
	if gssr.Title == "" {
		isSiteTitle = false
	}

	isSiteFavicon := true
	if gssr.Favicon == "" {
		isSiteFavicon = false
	}

	isSiteLogo := true
	if gssr.Logo == "" {
		isSiteLogo = false
	}

	var faviconExt string
	faviconSplit := strings.Split(gssr.Favicon, ".")
	if len(faviconSplit) > 1 {
		faviconExt = faviconSplit[1]
	}

	var isSiteLogoSVG bool
	var svgXML template.HTML
	var siteLogoExt string
	siteLogoSplit := strings.Split(gssr.Logo, ".")
	if len(siteLogoSplit) > 1 {
		siteLogoExt = siteLogoSplit[1]
	}
	if siteLogoExt == "svg" {
		isSiteLogoSVG = true
		dat, err := ioutil.ReadFile(filepath.Join(conf.Paths.UploadAssetPath, gssr.Logo))
		errorhandler.CheckError("Error on Reading svg logo file", err)

		svgXML = template.HTML(dat)
	}

	siteSettings := SiteSettings{
		IsSiteTitle:    isSiteTitle,
		IsSiteFavicon:  isSiteFavicon,
		IsSiteLogo:     isSiteLogo,
		SiteTitle:      gssr.Title,
		SiteFavicon:    gssr.Favicon,
		SiteFaviconExt: faviconExt,
		SiteLogo:       gssr.Logo,
		SiteLogoWidth:  gssr.LogoWidth,
		SiteLogoHeight: gssr.LogoHeight,
		IsSiteLogoSVG:  isSiteLogoSVG,
		SVGDAT:         svgXML,
	}

	return siteSettings
}
