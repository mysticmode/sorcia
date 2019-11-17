package handler

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	errorhandler "sorcia/error"

	"github.com/gorilla/mux"
)

// GitHandlerReqStruct struct
type GitHandlerReqStruct struct {
	w    http.ResponseWriter
	r    *http.Request
	RPC  string
	Dir  string
	File string
}

func applyGitHandlerReq(w http.ResponseWriter, r *http.Request, rpc, repoPath string) *GitHandlerReqStruct {
	vars := mux.Vars(r)
	username := vars["username"]
	reponame := vars["reponame"]

	URLPath := r.URL.Path
	file := strings.Split(URLPath, "/+"+username+"/"+reponame)[1]

	return &GitHandlerReqStruct{
		w:    w,
		r:    r,
		RPC:  rpc,
		Dir:  repoPath,
		File: file,
	}
}

func getServiceType(r *http.Request) string {
	vars := r.URL.Query()
	serviceType := vars["service"][0]
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
		ghrs.w.WriteHeader(http.StatusNotFound)
		return
	}

	ghrs.w.Header().Set("Content-Type", contentType)
	ghrs.w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	ghrs.w.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	http.ServeFile(ghrs.w, ghrs.r, reqFile)
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

func hdrNocache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	expires := now + 31536000
	w.Header().Set("Date", fmt.Sprintf("%d", now))
	w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}

// PostServiceRPC ...
func PostServiceRPC(w http.ResponseWriter, r *http.Request, db *sql.DB, repoPath string) {
	vars := mux.Vars(r)
	username := vars["username"]
	reponame := vars["reponame"]
	rpc := vars["rpc"]

	if rpc != "upload-pack" && rpc != "receive-pack" {
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	reqBody := r.Body

	// Handle GZIP
	if r.Header.Get("Content-Encoding") == "gzip" {
		_, err := gzip.NewReader(reqBody)
		if err != nil {
			errorhandler.CheckError(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(dir)
	repoDir := filepath.Join(repoPath, `\+`+username+`\`+reponame)
	fmt.Println(repoDir)

	cmd := exec.Command("git", rpc, "--stateless-rpc", repoDir)
	// if rpc == "receive-pack" {
	// 	username, password, authOK := c.Request.BasicAuth()
	// 	if authOK == false {
	// 		http.Error(w, "Not authorized", 401)
	// 		return
	// 	}

	// 	if username != "username" || password != "password" {
	// 		http.Error(w, "Not authorized", 401)
	// 		return
	// 	}
	// }

	// username, password, authOK := r.BasicAuth()
	// if authOK == false {
	// 	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	// 	w.WriteHeader(http.StatusUnauthorized)
	// 	w.Write([]byte("Unauthorised.\n"))
	// 	return
	// }

	// if username != "username" || password != "password" {
	// 	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	// 	w.WriteHeader(http.StatusUnauthorized)
	// 	w.Write([]byte("Not authorized.\n"))
	// 	return
	// }

	var stderr bytes.Buffer

	cmd.Dir = repoDir
	cmd.Stdout = w
	cmd.Stderr = &stderr
	cmd.Stdin = reqBody
	if err := cmd.Run(); err != nil {
		fmt.Println(fmt.Sprintf("Fail to serve RPC(%s): %v - %s", rpc, err, stderr.String()))
		return
	}
}

// GetInfoRefs ...
func GetInfoRefs(w http.ResponseWriter, r *http.Request, db *sql.DB, repoPath string) {
	vars := mux.Vars(r)
	username := vars["username"]
	reponame := vars["reponame"]

	hdrNocache(w)

	service := getServiceType(r)
	args := []string{service, "--stateless-rpc", "--advertise-refs", "."}

	// dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(dir)
	repoDir := filepath.Join(repoPath, `\+`+username+`\`+reponame)
	fmt.Println(repoDir)

	ghrs := applyGitHandlerReq(w, r, "", repoDir)

	if service != "upload-pack" && service != "receive-pack" {
		updateServerInfo(ghrs.Dir)
		ghrs.sendFile("text/plain; charset=utf-8")
		return
	}

	refs := gitCommand(ghrs.Dir, args...)
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))
	w.WriteHeader(http.StatusOK)
	w.Write(packetWrite("# service=git-" + service + "\n"))
	w.Write(packetFlush())
	w.Write(refs)
}

// GetGitRegexRequestHandler ...
func GetGitRegexRequestHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, repoPath string) {
	vars := mux.Vars(r)
	regex1 := vars["regex1"]
	regex2 := vars["regex2"]

	ghrs := applyGitHandlerReq(w, r, "", repoPath)

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
	hdrCacheForever(ghrs.w)
	ghrs.sendFile("text/plain; charset=utf-8")
}

// GetLooseObject ...
func (ghrs *GitHandlerReqStruct) GetLooseObject() {
	hdrCacheForever(ghrs.w)
	ghrs.sendFile("application/x-git-loose-object")
}

// GetPackFile ...
func (ghrs *GitHandlerReqStruct) GetPackFile() {
	hdrCacheForever(ghrs.w)
	ghrs.sendFile("application/x-git-packed-objects")
}

// GetIdxFile ...
func (ghrs *GitHandlerReqStruct) GetIdxFile() {
	hdrCacheForever(ghrs.w)
	ghrs.sendFile("application/x-git-packed-objects-toc")
}

// GetTextFile ...
func (ghrs *GitHandlerReqStruct) GetTextFile() {
	hdrNocache(ghrs.w)
	ghrs.sendFile("text/plain")
}

// GetHEADFile ...
func GetHEADFile(w http.ResponseWriter, r *http.Request, db *sql.DB, repoPath string) {
	hdrNocache(w)
	ghrs := applyGitHandlerReq(w, r, "", repoPath)
	ghrs.sendFile("text/plain")
}
