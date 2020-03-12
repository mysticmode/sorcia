package handler

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
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
	IsLoggedIn         bool
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
			IsLoggedIn:       true,
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
			IsLoggedIn:         true,
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
			IsLoggedIn:         true,
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
	gitPath := util.GetGitBinPath()

	args := []string{"init", "--bare", bareRepoDir}
	_ = util.ForkExec(gitPath, args, ".")

	http.Redirect(w, r, "/", http.StatusFound)
}

// GetRepoResponse struct
type GetRepoResponse struct {
	IsLoggedIn       bool
	ShowLoginMenu    bool
	HeaderActiveMenu string
	SorciaVersion    string
	Username         string
	Reponame         string
	RepoDescription  string
	IsRepoPrivate    bool
	Host             string
	TotalCommits     string
	TotalRefs        int
	RepoDetail       RepoDetail
	RepoLogs         RepoLogs
	RepoRefs         []Refs
	Contributors     Contributors
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

func checkUserLoggedIn(w http.ResponseWriter) bool {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		return true
	}

	return false
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	repoDir := filepath.Join(repoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userID := model.GetUserIDFromReponame(db, reponame)
	username := model.GetUsernameFromUserID(db, userID)
	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)
	totalCommits := util.GetCommitCounts(repoPath, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Username:         username,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		Host:             r.Host,
		TotalCommits:     totalCommits,
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	data.RepoDetail.Readme = processREADME(repoDir)

	commits := getCommits(repoPath, reponame, "master", "", -4)
	data.RepoLogs = *commits

	_, totalTags := util.GetGitTags(repoDir)
	data.TotalRefs = totalTags

	contributors := getContributors(repoPath, reponame, false)
	data.Contributors = *contributors

	writeRepoResponse(w, r, db, reponame, "repo-summary.html", data)
	return
}

func processREADME(repoPath string) template.HTML {

	gitPath := util.GetGitBinPath()
	args := []string{"show", "master:README.md"}

	out := util.ForkExec(gitPath, args, repoPath)

	md := []byte(out)
	output := blackfriday.Run(md)

	html := template.HTML(output)

	return html
}

// GetRepoTree ...
func GetRepoTree(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branch := vars["branch"]

	repoDir := filepath.Join(repoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gitPath := util.GetGitBinPath()

	dirs, files := walkThrough(repoDir, gitPath, branch, ".", 0)

	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.PathEmpty = true

	for _, dir := range dirs {
		repoDirDetail := RepoDirDetail{}
		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", dir}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoDirDetail.DirName = dir
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoDirDetail.DirCommit = commit
		repoDirDetail.DirCommitDate = ss[1]
		data.RepoDetail.RepoDirsDetail = append(data.RepoDetail.RepoDirsDetail, repoDirDetail)
	}

	for _, file := range files {
		repoFileDetail := RepoFileDetail{}
		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", file}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoFileDetail.FileName = file
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoFileDetail.FileCommit = commit
		repoFileDetail.FileCommitDate = ss[1]
		data.RepoDetail.RepoFilesDetail = append(data.RepoDetail.RepoFilesDetail, repoFileDetail)
	}

	commit := getCommits(repoPath, reponame, branch, "", -2)
	data.RepoLogs = *commit
	if len(data.RepoLogs.History) == 1 {
		data.RepoLogs.History[0].Message = util.LimitCharLengthInString(data.RepoLogs.History[0].Message)
	}

	writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
	return
}

// GetRepoTreePath ...
func GetRepoTreePath(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branch := vars["branch"]

	repoDir := filepath.Join(repoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	frdpath := strings.Split(r.URL.Path, "r/"+reponame+"/tree/"+branch+"/")[1]

	legendHref := "\"/r/" + reponame + "/tree/" + branch + "\""
	legendPath := "<a href=" + legendHref + ">" + reponame + "</a>"

	legendPathSplit := strings.Split(frdpath, "/")

	for _, s := range legendPathSplit {
		legendHref = strings.TrimSuffix(legendHref, "\"")
		legendHref = fmt.Sprintf("%s/%s\"", legendHref, s)

		additionalPath := "<a href=" + legendHref + ">" + s + "</a>"

		legendPath = fmt.Sprintf("%s / %s", legendPath, additionalPath)
	}

	data.RepoDetail.PathEmpty = false
	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.LegendPath = template.HTML(legendPath)

	gitPath := util.GetGitBinPath()
	frdPathLen := len(strings.Split(frdpath, "/"))
	dirs, files := walkThrough(repoDir, gitPath, branch, frdpath, frdPathLen)

	if len(dirs) == 0 && len(files) == 0 {
		args := []string{"show", fmt.Sprintf("%s:%s", branch, frdpath)}
		out := util.ForkExec(gitPath, args, repoDir)

		frdSplit := strings.Split(frdpath, "/")

		frdFile := frdSplit[len(frdSplit)-1]

		fileDotSplit := strings.Split(frdFile, ".")
		var fileContent string
		if len(fileDotSplit) > 1 {
			fileContent = fmt.Sprintf("<pre><code class=\"%s\">%s</code></pre>", fileDotSplit[1], out)
		} else {
			fileContent = fmt.Sprintf("<pre><code class=\"plaintext\">%s</code></pre>", out)
		}

		html := template.HTML(fileContent)

		data.RepoDetail.FileContent = html

		writeRepoResponse(w, r, db, reponame, "file-viewer.html", data)
		return
	}

	for _, dir := range dirs {
		dirPath := fmt.Sprintf("%s/%s", frdpath, dir)
		repoDirDetail := RepoDirDetail{}
		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", dirPath}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoDirDetail.DirName = dir
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoDirDetail.DirCommit = commit
		repoDirDetail.DirCommitDate = ss[1]
		data.RepoDetail.RepoDirsDetail = append(data.RepoDetail.RepoDirsDetail, repoDirDetail)
	}

	for _, file := range files {
		filePath := fmt.Sprintf("%s/%s", frdpath, file)
		repoFileDetail := RepoFileDetail{}
		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr", "--", filePath}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoFileDetail.FileName = file
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoFileDetail.FileCommit = commit
		repoFileDetail.FileCommitDate = ss[1]
		data.RepoDetail.RepoFilesDetail = append(data.RepoDetail.RepoFilesDetail, repoFileDetail)
	}

	commit := getCommits(repoPath, reponame, branch, "", -2)
	data.RepoLogs = *commit
	if len(data.RepoLogs.History) == 1 {
		data.RepoLogs.History[0].Message = util.LimitCharLengthInString(data.RepoLogs.History[0].Message)
	}

	writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
	return
}

// Walk through files and folders
func walkThrough(repoDir, gitPath, branch, lsTreePath string, lsTreePathLen int) ([]string, []string) {
	var dirs, files []string

	args := []string{"ls-tree", "-r", "--name-only", branch, "HEAD", lsTreePath + "/"}
	out := util.ForkExec(gitPath, args, repoDir)

	ss := strings.Split(out, "\n")
	entries := ss[:len(ss)-1]

	for _, entry := range entries {
		entrySplit := strings.Split(entry, "/")

		if len(entrySplit) == 1 {
			files = append(files, entrySplit[0])
		} else if lsTreePathLen == 0 {
			if !util.ContainsValueInArr(dirs, entrySplit[0]) {
				dirs = append(dirs, entrySplit[0])
			}
		} else {
			newPath := strings.Join(entrySplit[:lsTreePathLen+1], "/")
			args = []string{"ls-tree", "-r", "--name-only", branch, "HEAD", newPath}
			out = util.ForkExec(gitPath, args, repoDir)
			ss = strings.Split(out, "\n")
			newEntries := ss[:len(ss)-1]

			for _, newEntry := range newEntries {
				newEntrySplit := strings.Split(newEntry, "/")

				if len(newEntrySplit) == (lsTreePathLen + 1) {
					files = append(files, newEntrySplit[lsTreePathLen])
				} else {
					if !util.ContainsValueInArr(dirs, newEntrySplit[lsTreePathLen]) {
						dirs = append(dirs, newEntrySplit[lsTreePathLen])
					}
				}
			}
		}
	}

	return dirs, files
}

// GetRepoLog ...
func GetRepoLog(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branch := vars["branch"]

	q := r.URL.Query()
	qFrom := q["from"]

	var fromHash string

	if len(qFrom) > 0 {
		fromHash = qFrom[0]
	}

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	commits := getCommits(repoPath, reponame, branch, fromHash, -11)
	data.RepoLogs = *commits

	writeRepoResponse(w, r, db, reponame, "repo-log.html", data)
	return
}

type Refs struct {
	Version   string
	Targz     string
	TargzPath string
	Zip       string
	ZipPath   string
	Message   string
}

// GetRepoRefs ...
func GetRepoRefs(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath, refsPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	repoDir := filepath.Join(repoPath, reponame+".git")

	gitPath := util.GetGitBinPath()
	args := []string{"for-each-ref", "--sort=-taggerdate", "--format", "%(refname) %(contents:subject)", "refs/tags"}
	out := util.ForkExec(gitPath, args, repoDir)

	lineSplit := strings.Split(out, "\n")
	lines := lineSplit[:len(lineSplit)-1]

	var rfs []Refs

	for _, line := range lines {
		var rf Refs

		refFields := strings.Fields(line)

		rf.Version = strings.Split(refFields[0], "/")[2]

		rf.Message = strings.Join(refFields[1:], " ")

		tagname := rf.Version

		// Remove 'v' prefix from version
		if strings.HasPrefix(tagname, "v") {
			tagname = strings.Split(tagname, "v")[1]
		}

		// Generate tar.gz file
		tarFilename := fmt.Sprintf("%s-%s.tar.gz", reponame, tagname)
		tarRefPath := filepath.Join(refsPath, tarFilename)

		if _, err := os.Stat(tarRefPath); !os.IsNotExist(err) {
			rf.Targz = tarFilename
			rf.TargzPath = fmt.Sprintf("/dl/%s", tarFilename)
		}

		// Generate zip file
		zipFilename := fmt.Sprintf("%s-%s.zip", reponame, tagname)
		zipRefPath := filepath.Join(refsPath, zipFilename)

		if _, err := os.Stat(zipRefPath); !os.IsNotExist(err) {
			rf.Zip = zipFilename
			rf.ZipPath = fmt.Sprintf("/dl/%s", zipFilename)
		}

		rfs = append(rfs, rf)
	}

	data.RepoRefs = rfs

	writeRepoResponse(w, r, db, reponame, "repo-refs.html", data)
	return
}

func ServeRefFile(w http.ResponseWriter, r *http.Request, refsPath string) {
	vars := mux.Vars(r)
	fileName := vars["file"]
	dlPath := filepath.Join(refsPath, fileName)
	http.ServeFile(w, r, dlPath)
}

// GetRepoContributors ...
func GetRepoContributors(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion, repoPath string) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    sorciaVersion,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	contributors := getContributors(repoPath, reponame, true)

	data.Contributors = *contributors

	writeRepoResponse(w, r, db, reponame, "repo-contributors.html", data)
	return
}

type Contributors struct {
	Detail []Contributor
	Total  string
}

type Contributor struct {
	Name    string
	DP      string
	Commits string
}

func getContributors(repoPath, reponame string, getDetail bool) *Contributors {
	gitPath := util.GetGitBinPath()
	dirPath := filepath.Join(repoPath, reponame+".git")

	args := []string{"shortlog", "HEAD", "-sne"}
	out := util.ForkExec(gitPath, args, dirPath)

	cStringRmLastLn := strings.TrimSuffix(out, "\n")
	lines := strings.Split(cStringRmLastLn, "\n")

	var contributors Contributors

	contributors.Total = strconv.Itoa(len(lines))

	if getDetail {
		for _, line := range lines {
			lineDetail := strings.Fields(line)
			var contributor Contributor
			contributor.Commits = lineDetail[0]
			lineFurther := strings.Join(lineDetail[1:], " ")
			contributor.Name = strings.Split(lineFurther, " <")[0]
			emailSplit := strings.Split(lineFurther, " <")[1]
			email := strings.Split(emailSplit, ">")[0]

			hash := md5.Sum([]byte(email))
			stringHash := hex.EncodeToString(hash[:])
			contributor.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

			contributors.Detail = append(contributors.Detail, contributor)
		}
	}

	return &contributors
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
	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, reponame); !isRepoPrivate {
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

// RepoLogs struct
type RepoLogs struct {
	History  []RepoLog
	HashLink string
	IsNext   bool
}

func getCommits(repoPath, reponame, branch, fromHash string, commits int) *RepoLogs {
	dirPath := filepath.Join(repoPath, reponame+".git")

	ss := getGitCommits(commits, branch, fromHash, dirPath)

	rla := RepoLogs{}
	rl := RepoLog{}

	var hashLink string

	for i := 0; i < len(ss); i++ {
		if i == (len(ss) - 1) {
			hashLink = strings.Split(ss[i], "||srca-sptra||")[0]

			gitPath := util.GetGitBinPath()
			args := []string{"rev-list", "--max-parents=0", "HEAD"}
			out := util.ForkExec(gitPath, args, dirPath)

			lastHash := strings.Split(out, "\n")[0]

			if hashLink != lastHash {
				rla.IsNext = true
				break
			}
		}
		st := strings.Split(ss[i], "||srca-sptra||")
		rl.Hash = st[1]
		rl.Message = st[3]
		rl.Date = st[4]
		rl.Author = st[5]

		hash := md5.Sum([]byte(st[6]))
		stringHash := hex.EncodeToString(hash[:])
		rl.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

		rla = RepoLogs{
			History: append(rla.History, rl),
		}
	}

	rla.HashLink = hashLink

	return &rla
}

func getGitCommits(commits int, branch, fromHash, dirPath string) []string {
	gitPath := util.GetGitBinPath()

	var args []string
	if fromHash == "" {
		args = []string{"log", branch, strconv.Itoa(commits), "--pretty=format:%H||srca-sptra||%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae"}
	} else {
		args = []string{"log", branch, fromHash, strconv.Itoa(commits), "--pretty=format:%H||srca-sptra||%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae"}
	}
	out := util.ForkExec(gitPath, args, dirPath)

	ss := strings.Split(out, "\n")

	return ss
}
