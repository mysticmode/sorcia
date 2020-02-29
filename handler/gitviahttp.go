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
	"sorcia/model"
	"sorcia/util"
)

type gitHandler struct {
	w        http.ResponseWriter
	r        *http.Request
	rpc      string
	dir      string
	file     string
	reponame string
	db       *sql.DB
}

func (gh *gitHandler) basicAuth(username, passwordHash, realm string) bool {

	user, pass, ok := gh.r.BasicAuth()

	isPasswordValid := CheckPasswordHash(pass, passwordHash)

	if !ok || user != username || !isPasswordValid {
		gh.w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
		writeHdr(gh.w, http.StatusUnauthorized, "The repository cannot be accessed with your credentials.\n")
		return false
	}

	return true
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

func (gh *gitHandler) sendFile(contentType string) {
	reqFile := path.Join(gh.dir, gh.file)
	fi, err := os.Stat(reqFile)
	if os.IsNotExist(err) {
		gh.w.WriteHeader(http.StatusNotFound)
		return
	}

	gh.w.Header().Set("Content-Type", contentType)
	gh.w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	gh.w.Header().Set("Last-Modified", fi.ModTime().Format(http.TimeFormat))
	http.ServeFile(gh.w, gh.r, reqFile)
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

func (gh *gitHandler) hdrNocache() {
	gh.w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	gh.w.Header().Set("Pragma", "no-cache")
	gh.w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func (gh *gitHandler) hdrCacheForever() {
	now := time.Now().Unix()
	expires := now + 31536000
	gh.w.Header().Set("Date", fmt.Sprintf("%d", now))
	gh.w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	gh.w.Header().Set("Cache-Control", "public, max-age=31536000")
}

func serviceUploadPack(gh gitHandler) {
	postServiceRPC(gh, "upload-pack")
}

func serviceReceivePack(gh gitHandler) {
	postServiceRPC(gh, "receive-pack")
}

func postServiceRPC(gh gitHandler, rpc string) {
	rts := model.RepoTypeStruct{
		Reponame: gh.reponame,
	}

	// Check if repository is private
	if isRepoPrivate := model.GetRepoType(gh.db, &rts); isRepoPrivate {
		userID := model.GetUserIDFromReponame(gh.db, gh.reponame)
		if model.CheckRepoAccessFromUserID(gh.db, userID) {
			if processRepoAccess(gh) == false {
				return
			}
		}
	}

	if rpc == "receive-pack" && processRepoAccess(gh) == false {
		gh.w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if gh.r.Header.Get("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
		gh.w.WriteHeader(http.StatusUnauthorized)
		return
	}

	gh.w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))

	var err error
	reqBody := gh.r.Body

	// Handle GZIP
	if gh.r.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			fmt.Printf("Fail to create gzip reader: %v", err)
			gh.w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	cmd := exec.Command("git", rpc, "--stateless-rpc", gh.dir)

	var stderr bytes.Buffer

	cmd.Dir = gh.dir
	cmd.Stdin = reqBody
	cmd.Stdout = gh.w
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(fmt.Sprintf("Fail to serve RPC(%s): %v - %s", rpc, err, stderr.String()))
		return
	}

	if rpc == "receive-pack" {
		go util.PullFromAllBranches(gh.dir)
	}
}

func getInfoRefs(gh gitHandler) {
	gh.hdrNocache()

	rpc := getServiceType(gh.r)

	if rpc != "upload-pack" && rpc != "receive-pack" {
		gh := gitHandler{}
		updateServerInfo(gh.dir)
		gh.sendFile("text/plain; charset=utf-8")
		return
	}

	rts := model.RepoTypeStruct{
		Reponame: gh.reponame,
	}

	// Check if repository is private
	if isRepoPrivate := model.GetRepoType(gh.db, &rts); isRepoPrivate {
		userID := model.GetUserIDFromReponame(gh.db, gh.reponame)
		if model.CheckRepoAccessFromUserID(gh.db, userID) {
			if processRepoAccess(gh) == false {
				return
			}
		}
	}

	if rpc == "receive-pack" && processRepoAccess(gh) == false {
		return
	}

	refs := gitCommand(gh.dir, rpc, "--stateless-rpc", "--advertise-refs", ".")
	gh.w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", rpc))
	gh.w.WriteHeader(http.StatusOK)
	gh.w.Write(packetWrite("# service=git-" + rpc + "\n"))
	gh.w.Write([]byte("0000"))
	gh.w.Write(refs)

	if rpc == "receive-pack" {
		go util.PullFromAllBranches(gh.dir)
	}
}

func getTextFile(gh gitHandler) {
	gh.hdrNocache()
	gh.sendFile("text/plain")
}

func getInfoPacks(gh gitHandler) {
	gh.hdrCacheForever()
	gh.sendFile("text/plain; charset=utf-8")
}

func getLooseObject(gh gitHandler) {
	gh.hdrCacheForever()
	gh.sendFile("application/x-git-loose-object")
}

func getPackFile(gh gitHandler) {
	gh.hdrCacheForever()
	gh.sendFile("application/x-git-packed-objects")
}

func getIdxFile(gh gitHandler) {
	gh.hdrCacheForever()
	gh.sendFile("application/x-git-packed-objects-toc")
}

var routes = []struct {
	rxp     *regexp.Regexp
	method  string
	handler func(gitHandler)
}{
	{regexp.MustCompile("(.*?)/git-upload-pack$"), "POST", serviceUploadPack},
	{regexp.MustCompile("(.*?)/git-receive-pack$"), "POST", serviceReceivePack},
	{regexp.MustCompile("(.*?)/info/refs$"), "GET", getInfoRefs},
	{regexp.MustCompile("(.*?)/HEAD$"), "GET", getTextFile},
	{regexp.MustCompile("(.*?)/objects/info/alternates$"), "GET", getTextFile},
	{regexp.MustCompile("(.*?)/objects/info/http-alternates$"), "GET", getTextFile},
	{regexp.MustCompile("(.*?)/objects/info/packs$"), "GET", getInfoPacks},
	{regexp.MustCompile("(.*?)/objects/info/[^/]*$"), "GET", getTextFile},
	{regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$"), "GET", getLooseObject},
	{regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$"), "GET", getPackFile},
	{regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$"), "GET", getIdxFile},
}

func writeHdr(w http.ResponseWriter, status int, text string) {
	w.WriteHeader(status)
	_, err := w.Write([]byte(text))
	errorhandler.CheckError(err)
}

func getProjectRootDir() string {
	projectRootDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	errorhandler.CheckError(err)
	return projectRootDir
}

func processRepoAccess(gh gitHandler) bool {
	userID := model.GetUserIDFromReponame(gh.db, gh.reponame)
	username := model.GetUsernameFromUserID(gh.db, userID)

	sphjwt := model.SelectPasswordHashAndJWTTokenStruct{
		Username: username,
	}
	sphjwtr := model.SelectPasswordHashAndJWTToken(gh.db, sphjwt)
	passwordHash := sphjwtr.PasswordHash

	if gh.basicAuth(username, passwordHash, "Please enter your username and password") {
		return true
	}

	return false
}

// GitviaHTTP ...
func GitviaHTTP(w http.ResponseWriter, r *http.Request, db *sql.DB, dir string) {
	for _, route := range routes {
		reqPath := strings.ToLower(r.URL.Path)
		reqPath = "/" + strings.Split(reqPath, "/r/")[1]
		routeMatch := route.rxp.FindStringSubmatch(reqPath)

		if routeMatch == nil {
			continue
		}

		if route.method != r.Method {
			if r.Proto == "HTTP/1.1" {
				writeHdr(w, http.StatusMethodNotAllowed, "Method not allowed")
			} else {
				writeHdr(w, http.StatusBadRequest, "Bad request")
			}
			return
		}

		var repoDir string
		projectRootDir := getProjectRootDir()

		if dir == "." || dir == "" || dir == "./repositories" {
			repoDir = filepath.Join(projectRootDir, "repositories", routeMatch[1])
		} else {
			repoDir = filepath.Join(dir, routeMatch[1])
		}

		file := strings.TrimPrefix(reqPath, routeMatch[1]+"/")
		reponame := strings.TrimSuffix(strings.TrimPrefix(routeMatch[1], "/"), ".git")

		gh := gitHandler{
			w:        w,
			r:        r,
			dir:      repoDir,
			file:     file,
			reponame: reponame,
			db:       db,
		}

		route.handler(gh)

		return
	}

	writeHdr(w, http.StatusNotFound, "Not found")
}
