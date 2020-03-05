package handler

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/util"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/russross/blackfriday/v2"
)

// GetCreateRepoResponse struct
type GetCreateRepoResponse struct {
	IsHeaderLogin      bool
	HeaderActiveMenu   string
	ReponameErrMessage string
	SorciaVersion      string
}

// GetCreateRepo ...
func GetCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		createRepoPage := path.Join("./templates", "create-repo.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := GetCreateRepoResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "",
			SorciaVersion:    sorciaVersion,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `schema:"name"`
	Description string `schema:"description"`
	IsPrivate   string `schema:"is_private"`
}

// PostCreateRepo ...
func PostCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, decoder *schema.Decoder, sorciaVersion, repoPath string) {
	// NOTE: Invoke ParseForm or ParseMultipartForm before reading form values
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		errorResponse := &errorhandler.Response{
			Error: err.Error(),
		}

		errorJSON, err := json.Marshal(errorResponse)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		w.Write(errorJSON)
	}

	var createRepoRequest = &CreateRepoRequest{}
	err := decoder.Decode(createRepoRequest, r.PostForm)
	errorhandler.CheckError(err)

	s := createRepoRequest.Name
	if len(s) > 100 || len(s) < 1 {
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		createRepoPage := path.Join("./templates", "create-repo.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := GetCreateRepoResponse{
			IsHeaderLogin:      false,
			HeaderActiveMenu:   "",
			ReponameErrMessage: "Repository name is too long (maximum is 100 characters).",
			SorciaVersion:      sorciaVersion,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
		return
	} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !util.IsAlnumOrHyphen(s) {
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		createRepoPage := path.Join("./templates", "create-repo.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := GetCreateRepoResponse{
			IsHeaderLogin:      false,
			HeaderActiveMenu:   "",
			ReponameErrMessage: "Repository name may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
			SorciaVersion:      sorciaVersion,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
		return
	}

	token := w.Header().Get("sorcia-cookie-token")

	userID := model.GetUserIDFromToken(db, token)

	var isPrivate int
	if isPrivate = 0; createRepoRequest.IsPrivate == "1" {
		isPrivate = 1
	}

	crs := model.CreateRepoStruct{
		Name:        createRepoRequest.Name,
		Description: createRepoRequest.Description,
		IsPrivate:   isPrivate,
		UserID:      userID,
	}

	model.InsertRepo(db, crs)

	// Create Git bare repository
	bareRepoDir := filepath.Join(repoPath, createRepoRequest.Name+".git")

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	err = cmd.Run()
	errorhandler.CheckError(err)

	// Clone from the bare repository created above
	repoDir := filepath.Join(repoPath, createRepoRequest.Name)
	cmd = exec.Command("git", "clone", bareRepoDir, repoDir)
	err = cmd.Run()
	errorhandler.CheckError(err)

	http.Redirect(w, r, "/", http.StatusFound)
}

// GetRepoResponse struct
type GetRepoResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	SorciaVersion    string
	Username         string
	Reponame         string
	RepoDescription  string
	IsRepoPrivate    bool
	Host             string
	RepoDetail       RepoDetail
	RepoLogs         RepoLogs
}

// RepoDetail struct
type RepoDetail struct {
	Readme          template.HTML
	FileContent     template.HTML
	LegendPath      template.HTML
	WalkPath        string
	PathEmpty       bool
	RepoDirsDetail  []RepoDirDetail
	RepoFilesDetail []RepoFileDetail
}

// RepoDirDetail struct
type RepoDirDetail struct {
	DirName       string
	DirCommit     string
	DirCommitDate string
}

// RepoFileDetail struct
type RepoFileDetail struct {
	FileName       string
	FileCommit     string
	FileCommitDate string
}

// RepoLog struct
type RepoLog struct {
	Hash    string
	Author  string
	Date    string
	Message string
	DP      string
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userID := model.GetUserIDFromReponame(db, reponame)
	username := model.GetUsernameFromUserID(db, userID)
	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Username:         username,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    false,
		Host:             r.Host,
	}

	data.RepoDetail.Readme = processREADME(repoPath, reponame)

	commits := getCommits(repoPath, reponame, -3)
	data.RepoLogs = *commits

	writeRepoResponse(w, r, db, reponame, "repo-summary.html", data)
	return
}

func processREADME(repoPath, repoName string) template.HTML {
	readmeFile := filepath.Join(repoPath, repoName, "README.md")
	_, err := os.Stat(readmeFile)

	html := template.HTML("")

	if !os.IsNotExist(err) {
		dat, err := ioutil.ReadFile(readmeFile)
		if err != nil {
			fmt.Println(err)
		}
		md := []byte(dat)
		output := blackfriday.Run(md)

		html = template.HTML(output)
	}

	return html
}

// GetRepoTree ...
func GetRepoTree(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    false,
	}

	gitPath := util.GetGitBinPath()

	dirPath := filepath.Join(repoPath, reponame)
	dirs, files := walkThrough(dirPath, gitPath)

	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.PathEmpty = true

	for _, dir := range dirs {
		repoDirDetail := RepoDirDetail{}
		cmd := exec.Command(gitPath, "log", "master", "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", dir)
		cmd.Dir = dirPath

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}

		ss := strings.Split(out.String(), "||srca-sptra||")

		repoDirDetail.DirName = dir
		repoDirDetail.DirCommit = ss[0]
		repoDirDetail.DirCommitDate = ss[1]
		data.RepoDetail.RepoDirsDetail = append(data.RepoDetail.RepoDirsDetail, repoDirDetail)
	}

	for _, file := range files {
		repoFileDetail := RepoFileDetail{}
		cmd := exec.Command(gitPath, "log", "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", file)
		cmd.Dir = dirPath

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}

		ss := strings.Split(out.String(), "||srca-sptra||")

		repoFileDetail.FileName = file
		repoFileDetail.FileCommit = ss[0]
		repoFileDetail.FileCommitDate = ss[1]
		data.RepoDetail.RepoFilesDetail = append(data.RepoDetail.RepoFilesDetail, repoFileDetail)
	}

	writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
	return
}

// GetRepoTreePath ...
func GetRepoTreePath(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    false,
	}

	frdpath := strings.Split(r.URL.Path, "r/"+reponame+"/tree/")[1]

	dirPath := filepath.Join(repoPath, reponame, frdpath)

	legendPathSplit := strings.Split(frdpath, "/")

	legendPathArr := make([]string, len(legendPathSplit))

	for i, s := range legendPathSplit {
		if i == 0 {
			legendPathArr[i] = "<a href=\"/r/" + reponame + "/tree\">" + reponame + "</a> / <a href=\"/r/" + reponame + "/tree"
		} else {
			legendPathArr[i] = "<a href=\"/r/" + reponame + "/tree"
		}
		for j := 0; j <= i; j++ {
			legendPathArr[i] = fmt.Sprintf("%s/%s", legendPathArr[i], legendPathSplit[j])
		}
		legendPathArr[i] = fmt.Sprintf("%s\">%s</a>", legendPathArr[i], s)
	}

	fi, err := os.Stat(dirPath)
	errorhandler.CheckError(err)

	if fi.Mode().IsRegular() {
		file, err := os.Open(dirPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		var codeLines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			codeLines = append(codeLines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		code := strings.Join(codeLines, "\n")

		fileDotSplit := strings.Split(dirPath, ".")
		var fileExt string
		if len(fileDotSplit) > 1 {
			fileExt = fileDotSplit[len(fileDotSplit)-1]
		} else {
			fileExt = ""
		}

		code = template.HTMLEscaper(code)

		var fileContent string
		if fileExt == "" {
			fileContent = fmt.Sprintf("<pre><code class=\"plaintext\">%s</code></pre>", code)
		} else {
			fileContent = fmt.Sprintf("<pre><code class=\"%s\">%s</code></pre>", fileExt, code)
		}
		html := template.HTML(fileContent)

		legendPath := template.HTML(strings.Join(legendPathArr, " / "))

		data.RepoDetail.PathEmpty = false
		data.RepoDetail.WalkPath = r.URL.Path
		data.RepoDetail.LegendPath = legendPath
		data.RepoDetail.FileContent = html

		writeRepoResponse(w, r, db, reponame, "file-viewer.html", data)
		return
	}

	gitPath := util.GetGitBinPath()
	dirs, files := walkThrough(dirPath, gitPath)

	legendPath := template.HTML(strings.Join(legendPathArr, " / "))

	data.RepoDetail.PathEmpty = false
	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.LegendPath = legendPath

	for _, dir := range dirs {
		repoDirDetail := RepoDirDetail{}
		cmd := exec.Command(gitPath, "log", "master", "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", dir)
		cmd.Dir = dirPath

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}

		ss := strings.Split(out.String(), "||srca-sptra||")

		repoDirDetail.DirName = dir
		repoDirDetail.DirCommit = ss[0]
		repoDirDetail.DirCommitDate = ss[1]
		data.RepoDetail.RepoDirsDetail = append(data.RepoDetail.RepoDirsDetail, repoDirDetail)
	}

	for _, file := range files {
		repoFileDetail := RepoFileDetail{}
		cmd := exec.Command(gitPath, "log", "master", "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", file)
		cmd.Dir = dirPath

		var out, stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			fmt.Println(stderr.String())
		}

		ss := strings.Split(out.String(), "||srca-sptra||")

		repoFileDetail.FileName = file
		repoFileDetail.FileCommit = ss[0]
		repoFileDetail.FileCommitDate = ss[1]
		data.RepoDetail.RepoFilesDetail = append(data.RepoDetail.RepoFilesDetail, repoFileDetail)
	}

	writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
	return
}

// GetRepoLog ...
func GetRepoLog(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    false,
	}

	// commitCounts := getCommitCounts(repoPath, reponame)

	commits := getCommits(repoPath, reponame, -10)
	data.RepoLogs = *commits

	writeRepoResponse(w, r, db, reponame, "repo-log.html", data)
	return
}

// Walk through files and folders
func walkThrough(dirPath, gitPath string) ([]string, []string) {
	var dirs, files []string

	cmd := exec.Command(gitPath, "ls-tree", "--name-only", "master", "HEAD", ".")
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
	}

	ss := strings.Split(out.String(), "\n")
	entries := ss[:len(ss)-1]

	for _, entry := range entries {
		fi, err := os.Stat(filepath.Join(dirPath, entry))
		errorhandler.CheckError(err)

		if fi.Mode().IsRegular() {
			files = append(files, entry)
		} else {
			dirs = append(dirs, entry)
		}
	}

	return dirs, files
}

func noRepoAccess(w http.ResponseWriter) {
	errorResponse := &errorhandler.Response{
		Error: "You don't have access to this repository.",
	}

	errorJSON, err := json.Marshal(errorResponse)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	w.Write(errorJSON)
}

func writeRepoResponse(w http.ResponseWriter, r *http.Request, db *sql.DB, reponame string, mainPage string, data GetRepoResponse) {
	rts := model.RepoTypeStruct{
		Reponame: reponame,
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		tmpl := parseTemplates(w, mainPage)
		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent != "" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserIDAndReponame(db, userIDFromToken, reponame); hasRepoAccess {
				data.IsRepoPrivate = true
				tmpl := parseTemplates(w, mainPage)
				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

func parseTemplates(w http.ResponseWriter, mainPage string) *template.Template {
	layoutPage := path.Join("./templates", "layout.html")
	headerPage := path.Join("./templates", "header.html")
	repoLogPage := path.Join("./templates", mainPage)
	footerPage := path.Join("./templates", "footer.html")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, repoLogPage, footerPage)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	return tmpl
}

func getCommitCounts(repoPath, reponame string) string {
	dirPath := filepath.Join(repoPath, reponame+".git")
	cmd := exec.Command("/bin/git", "rev-list", "HEAD", "--count")
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
	}

	return strings.TrimSpace(out.String())
}

// RepoLogs struct
type RepoLogs struct {
	History []RepoLog
}

func getCommits(repoPath, reponame string, commits int) *RepoLogs {
	dirPath := filepath.Join(repoPath, reponame+".git")

	ss := getGitCommits(commits, dirPath)

	rla := RepoLogs{}
	rl := RepoLog{}

	for i := 0; i < len(ss); i++ {
		st := strings.Split(ss[i], "||srca-sptra||")
		rl.Hash = st[0]
		rl.Message = st[2]
		rl.Date = st[3]
		rl.Author = st[4]

		hash := md5.Sum([]byte(st[5]))
		stringHash := hex.EncodeToString(hash[:])
		rl.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

		rla = RepoLogs{
			History: append(rla.History, rl),
		}
	}

	return &rla
}

func getGitCommits(commits int, dirPath string) []string {
	gitPath := util.GetGitBinPath()

	cmd := exec.Command(gitPath, "log", strconv.Itoa(commits), "--pretty=format:%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae")
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
	}

	ss := strings.Split(out.String(), "\n")

	return ss
}
