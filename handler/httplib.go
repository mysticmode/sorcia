package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	errorhandler "sorcia/error"
	"sorcia/setting"

	"github.com/gin-gonic/gin"
)

// GitHandlerReqStruct struct
type GitHandlerReqStruct struct {
	c    *gin.Context
	RPC  string
	Dir  string
	File string
}

func applyGitHandlerReq(c *gin.Context, rpc string) *GitHandlerReqStruct {
	username := c.Param("username")
	reponame := c.Param("reponame")

	// Get config values
	conf := setting.GetConf()

	repoDir := path.Join(conf.Paths.DataPath, "repositories"+"/"+username+"/"+reponame)

	URLPath := c.Request.URL.Path
	file := strings.Split(URLPath, "/~"+username+"/"+reponame)[1]

	return &GitHandlerReqStruct{
		c:    c,
		RPC:  rpc,
		Dir:  repoDir,
		File: file,
	}
}

func getServiceType(c *gin.Context) string {
	q := c.Request.URL.Query()
	serviceType := q["service"][0]
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.TrimPrefix(serviceType, "git-")
}

func gitCommand(dir string, args ...string) []byte {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	errorhandler.CheckError(err)

	return out
}

func updateServerInfo(dir string) []byte {
	return gitCommand(dir, "update-server-info")
}

func (ghrs *GitHandlerReqStruct) sendFile(contentType string) {
	reqFile := path.Join(ghrs.Dir, ghrs.File)
	fi, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		ghrs.c.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	ghrs.c.Writer.Header().Set("Content-Type", contentType)
	ghrs.c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	ghrs.c.Writer.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	ghrs.c.File(reqFile)
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}

func packetFlush() []byte {
	return []byte("0000")
}

func hdrNocache(c *gin.Context) {
	c.Writer.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.Writer.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(c *gin.Context) {
	now := time.Now().Unix()
	expires := now + 31536000
	c.Writer.Header().Set("Date", fmt.Sprintf("%d", now))
	c.Writer.Header().Set("Expires", fmt.Sprintf("%d", expires))
	c.Writer.Header().Set("Cache-Control", "public, max-age=31536000")
}

// PostServiceRPC ...
func PostServiceRPC(c *gin.Context) {
	username := c.Param("username")
	reponame := c.Param("reponame")
	rpc := c.Param("rpc")

	if rpc != "upload-pack" && rpc != "receive-pack" {
		return
	}

	c.Writer.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	c.Writer.Header().Set("Connection", "Keep-Alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(http.StatusOK)

	reqBody := c.Request.Body

	// Handle GZIP
	if c.Request.Header.Get("Content-Encoding") == "gzip" {
		_, err := gzip.NewReader(reqBody)
		if err != nil {
			errorhandler.CheckError(err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Get config values
	conf := setting.GetConf()

	repoDir := (path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+reponame))

	cmd := exec.Command("git", rpc, "--stateless-rpc", repoDir)
	// if rpc == "receive-pack" {
	// 	username, password, authOK := c.Request.BasicAuth()
	// 	if authOK == false {
	// 		http.Error(c.Writer, "Not authorized", 401)
	// 		return
	// 	}

	// 	if username != "username" || password != "password" {
	// 		http.Error(c.Writer, "Not authorized", 401)
	// 		return
	// 	}
	// }

	username, password, authOK := c.Request.BasicAuth()
	if authOK == false {
		c.Writer.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		c.Writer.WriteHeader(401)
		c.Writer.Write([]byte("Unauthorised.\n"))
		return
	}

	if username != "username" || password != "password" {
		c.Writer.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		c.Writer.WriteHeader(401)
		c.Writer.Write([]byte("Not authorized.\n"))
		return
	}

	var stderr bytes.Buffer

	cmd.Dir = repoDir
	cmd.Stdout = c.Writer
	cmd.Stderr = &stderr
	cmd.Stdin = reqBody
	err := cmd.Run()
	errorhandler.CheckError(err)
	return
}

// GetInfoRefs ...
func GetInfoRefs(c *gin.Context) {
	hdrNocache(c)

	service := getServiceType(c)
	args := []string{service, "--stateless-rpc", "--advertise-refs", "."}

	ghrs := applyGitHandlerReq(c, "")

	if service != "upload-pack" && service != "receive-pack" {
		updateServerInfo(ghrs.Dir)
		ghrs.sendFile("text/plain; charset=utf-8")
		return
	}

	refs := gitCommand(ghrs.Dir, args...)
	c.Writer.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(packetWrite("# service=git-" + service + "\n"))
	c.Writer.Write(packetFlush())
	c.Writer.Write(refs)
}

// GetGitRegexRequestHandler ...
func GetGitRegexRequestHandler(c *gin.Context) {
	regex1 := c.Param("regex1")
	regex2 := c.Param("regex2")

	ghrs := applyGitHandlerReq(c, "")

	if regex1 == "info" {
		if regex2 == "alternates" || regex2 == "http-alternates" {
			ghrs.GetTextFile()
		} else if regex2 == "packs" {
			ghrs.GetInfoPacks()
		} else if match, _ := regexp.MatchString("[^/]*$", regex2); match {
			ghrs.GetTextFile()
		}
	} else if regex1 == "pack" {
		if match, _ := regexp.MatchString("pack-[0-9a-f]{40}\\.pack$", regex2); match {
			ghrs.GetPackFile()
		} else if match, _ = regexp.MatchString("pack-[0-9a-f]{40}\\.idx$", regex2); match {
			ghrs.GetIdxFile()
		}
	} else if match, _ := regexp.MatchString("[0-9a-f]{2}", regex1); match {
		if match, _ = regexp.MatchString("[0-9a-f]{38}$", regex2); match {
			ghrs.GetLooseObject()
		}
	}
}

// GetInfoPacks ...
func (ghrs *GitHandlerReqStruct) GetInfoPacks() {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("text/plain; charset=utf-8")
}

// GetLooseObject ...
func (ghrs *GitHandlerReqStruct) GetLooseObject() {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-loose-object")
}

// GetPackFile ...
func (ghrs *GitHandlerReqStruct) GetPackFile() {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-packed-objects")
}

// GetIdxFile ...
func (ghrs *GitHandlerReqStruct) GetIdxFile() {
	hdrCacheForever(ghrs.c)
	ghrs.sendFile("application/x-git-packed-objects-toc")
}

// GetTextFile ...
func (ghrs *GitHandlerReqStruct) GetTextFile() {
	hdrNocache(ghrs.c)
	ghrs.sendFile("text/plain")
}

// GetHEADFile ...
func GetHEADFile(c *gin.Context) {
	hdrNocache(c)
	ghrs := applyGitHandlerReq(c, "")
	ghrs.sendFile("text/plain")
}
