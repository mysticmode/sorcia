package handler

import (
	"bufio"
	"bytes"
	"database/sql"
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

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"gopkg.in/russross/blackfriday.v2"
)

// GetCreateRepoResponse struct
type GetCreateRepoResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	SorciaVersion    string
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
func PostCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, decoder *schema.Decoder, repoPath string) {
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
	IsRepoPrivate    bool
	Host             string
	RepoDetail       RepoDetail
	RepoLogs         RepoLogs
	DisplayCommits   string
}

// RepoDetail struct
type RepoDetail struct {
	Readme      template.HTML
	FileContent template.HTML
	LegendPath  template.HTML
	WalkPath    string
	PathEmpty   bool
	RepoDirs    []string
	RepoFiles   []string
}

// RepoLog struct
type RepoLog struct {
	IsHead  bool
	Hash    string
	Author  string
	Date    string
	Message string
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rts := model.RepoTypeStruct{
		Reponame: reponame,
	}

	userID := model.GetUserIDFromReponame(db, reponame)
	username := model.GetUsernameFromUserID(db, userID)

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Username:         username,
		Reponame:         reponame,
		IsRepoPrivate:    false,
		Host:             r.Host,
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		repoSummaryPage := path.Join("./templates", "repo-summary.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, repoSummaryPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data.RepoDetail.Readme = processREADME(repoPath, reponame)

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent == "true" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				data.IsRepoPrivate = true

				layoutPage := path.Join("./templates", "layout.html")
				headerPage := path.Join("./templates", "header.html")
				repoSummaryPage := path.Join("./templates", "repo-summary.html")
				footerPage := path.Join("./templates", "footer.html")

				tmpl, err := template.ParseFiles(layoutPage, headerPage, repoSummaryPage, footerPage)
				errorhandler.CheckError(err)

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)

				data.RepoDetail.Readme = processREADME(repoPath, reponame)

				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
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

	rts := model.RepoTypeStruct{
		Reponame: reponame,
	}

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		IsRepoPrivate:    false,
	}

	dirPath := filepath.Join(repoPath, reponame)
	dirs, files := walkThrough(dirPath)

	data.RepoDetail.RepoDirs = dirs
	data.RepoDetail.RepoFiles = files
	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.PathEmpty = true

	layoutPage := path.Join("./templates", "layout.html")
	headerPage := path.Join("./templates", "header.html")
	repoTreePage := path.Join("./templates", "repo-tree.html")
	footerPage := path.Join("./templates", "footer.html")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, repoTreePage, footerPage)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent != "" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				data.IsRepoPrivate = true
				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

// GetRepoTreePath ...
func GetRepoTreePath(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rts := model.RepoTypeStruct{
		Reponame: reponame,
	}

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		IsRepoPrivate:    false,
	}

	frdpath := strings.Split(r.URL.Path, "r/"+reponame+"/tree/")[1]

	dirPath := filepath.Join(repoPath, reponame, frdpath)
	fi, err := os.Stat(dirPath)
	errorhandler.CheckError(err)

	legendPathSplit := strings.Split(frdpath, "/")

	legendPathArr := make([]string, len(legendPathSplit))

	for i, s := range legendPathSplit {
		if i == 0 {
			legendPathArr[i] = "<a href=\"/r/sorcia/tree\">sorcia</a> / <a href=\"/r/" + reponame + "/tree"
		} else {
			legendPathArr[i] = "<a href=\"/r/" + reponame + "/tree"
		}
		for j := 0; j <= i; j++ {
			legendPathArr[i] = fmt.Sprintf("%s/%s", legendPathArr[i], legendPathSplit[j])
		}
		legendPathArr[i] = fmt.Sprintf("%s\">%s</a>", legendPathArr[i], s)
	}

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
		fileExt := fileDotSplit[len(fileDotSplit)-1]

		if fileExt == "html" || fileExt == "tmpl" || fileExt == "svg" {
			code = template.HTMLEscaper(code)
		}

		fileContent := fmt.Sprintf("<pre><code class=\"%s\">%s</code></pre>", fileExt, code)
		html := template.HTML(fileContent)

		legendPath := template.HTML(strings.Join(legendPathArr, " / "))

		// data.RepoDetail.RepoDirs = dirs
		// data.RepoDetail.RepoFiles = files
		data.RepoDetail.PathEmpty = false
		data.RepoDetail.WalkPath = r.URL.Path
		data.RepoDetail.LegendPath = legendPath
		data.RepoDetail.FileContent = html

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		fileViewerPage := path.Join("./templates", "file-viewer.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, fileViewerPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		tmpl.ExecuteTemplate(w, "layout", data)

		return
	}

	dirs, files := walkThrough(dirPath)

	legendPath := template.HTML(strings.Join(legendPathArr, " / "))

	data.RepoDetail.RepoDirs = dirs
	data.RepoDetail.RepoFiles = files
	data.RepoDetail.PathEmpty = false
	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.LegendPath = legendPath

	layoutPage := path.Join("./templates", "layout.html")
	headerPage := path.Join("./templates", "header.html")
	repoTreePage := path.Join("./templates", "repo-tree.html")
	footerPage := path.Join("./templates", "footer.html")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, repoTreePage, footerPage)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent != "" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				data.IsRepoPrivate = true
				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}

}

// Walk through files and folders
func walkThrough(dirPath string) ([]string, []string) {
	var dirs, files []string
	entries, err := ioutil.ReadDir(dirPath)
	errorhandler.CheckError(err)
	for _, f := range entries {
		if f.IsDir() && f.Name() != ".git" {
			dirs = append(dirs, f.Name())
		} else {
			if f.Name() != ".git" {
				files = append(files, f.Name())
			}
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

// GetRepoLog ...
func GetRepoLog(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rts := model.RepoTypeStruct{
		Reponame: reponame,
	}

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		IsRepoPrivate:    false,
	}

	commitCounts := getCommitCounts(repoPath, reponame)
	fmt.Println(commitCounts)

	commits := getCommits(repoPath, reponame, -10)
	data.RepoLogs = *commits

	layoutPage := path.Join("./templates", "layout.html")
	headerPage := path.Join("./templates", "header.html")
	repoLogPage := path.Join("./templates", "repo-log.html")
	footerPage := path.Join("./templates", "footer.html")

	tmpl, err := template.ParseFiles(layoutPage, headerPage, repoLogPage, footerPage)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent != "" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				data.IsRepoPrivate = true
				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
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

type RepoLogs struct {
	History []RepoLog
}

func getCommits(repoPath, reponame string, commits int) *RepoLogs {
	dirPath := filepath.Join(repoPath, reponame+".git")
	cmd := exec.Command("/usr/bin/git", "log", strconv.Itoa(commits), "--pretty=format:%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%ae")
	cmd.Dir = dirPath

	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
	}

	ss := strings.Split(out.String(), "\n")
	fmt.Println(out.String())
	fmt.Println(ss)

	rla := RepoLogs{}
	rl := RepoLog{}

	for i := 0; i < len(ss); i++ {
		st := strings.Split(ss[i], "||srca-sptra||")
		if strings.HasPrefix(strings.TrimSpace(st[1]), "(HEAD ->") {
			rl.IsHead = true
		} else {
			rl.IsHead = false
		}
		rl.Hash = st[0]
		rl.Message = st[2]
		rl.Date = st[3]
		rl.Author = st[4]

		rla = RepoLogs{
			History: append(rla.History, rl),
		}
	}

	return &rla
}
