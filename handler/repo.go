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
	"sorcia/setting"
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
	SiteSettings       util.SiteSettings
}

// GetCreateRepo ...
func GetCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		if !model.CheckifUserCanCreateRepo(db, userID) {
			http.Redirect(w, r, "/", http.StatusFound)
		}

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		createRepoPage := path.Join("./templates", "create-repo.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := GetCreateRepoResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "",
			SorciaVersion:    conf.Version,
			SiteSettings:     util.GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `schema:"name"`
	Description string `schema:"description"`
	IsPrivate   string `schema:"is_private"`
}

// PostCreateRepo ...
func PostCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, decoder *schema.Decoder, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)

		if !model.CheckifUserCanCreateRepo(db, userID) {
			http.Redirect(w, r, "/", http.StatusFound)
		}

		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &errorhandler.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			errorhandler.CheckError("Error on post create repo json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		var createRepoRequest = &CreateRepoRequest{}
		err := decoder.Decode(createRepoRequest, r.PostForm)
		errorhandler.CheckError("Error on post create repo decoder", err)

		s := createRepoRequest.Name
		if len(s) > 100 || len(s) < 1 {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			createRepoPage := path.Join("./templates", "create-repo.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := GetCreateRepoResponse{
				IsLoggedIn:         true,
				HeaderActiveMenu:   "",
				ReponameErrMessage: "Repository name is too long (maximum is 100 characters).",
				SorciaVersion:      conf.Version,
				SiteSettings:       util.GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !util.IsAlnumOrHyphen(s) {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			createRepoPage := path.Join("./templates", "create-repo.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, createRepoPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := GetCreateRepoResponse{
				IsLoggedIn:         true,
				HeaderActiveMenu:   "",
				ReponameErrMessage: "Repository name may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
				SorciaVersion:      conf.Version,
				SiteSettings:       util.GetSiteSettings(db, conf),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}

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
		bareRepoDir := filepath.Join(conf.Paths.RepoPath, createRepoRequest.Name+".git")
		gitPath := util.GetGitBinPath()

		args := []string{"init", "--bare", bareRepoDir}
		_ = util.ForkExec(gitPath, args, ".")

		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// GetRepoResponse struct
type GetRepoResponse struct {
	SiteSettings       util.SiteSettings
	SiteStyle          string
	IsLoggedIn         bool
	ShowLoginMenu      bool
	HeaderActiveMenu   string
	SorciaVersion      string
	Username           string
	RepoUserAddError   string
	Reponame           string
	ReponameErrMessage string
	RepoDescription    string
	IsRepoPrivate      bool
	RepoAccess         bool
	RepoMembers        model.GetRepoMembersStruct
	Host               string
	SSHClone           string
	TotalCommits       string
	TotalRefs          int
	RepoDetail         RepoDetail
	RepoBranches       []string
	IsRepoBranch       bool
	RepoLogs           RepoLogs
	CommitDetail       CommitDetailStruct
	RepoRefs           []Refs
	Contributors       Contributors
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
	DirName           string
	DirCommit         string
	DirCommitDate     string
	DirCommitFullHash string
	DirCommitBranch   string
}

// RepoFileDetail struct
type RepoFileDetail struct {
	FileName           string
	FileCommit         string
	FileCommitDate     string
	FileCommitFullHash string
	FileCommitBranch   string
}

// RepoLog struct
type RepoLog struct {
	FullHash string
	Hash     string
	Author   string
	Date     string
	Message  string
	DP       string
	Branch   string
}

func checkUserLoggedIn(w http.ResponseWriter) bool {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		return true
	}

	return false
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	userID := model.GetUserIDFromReponame(db, reponame)
	username := model.GetUsernameFromUserID(db, userID)
	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)
	totalCommits := util.GetCommitCounts(conf.Paths.RepoPath, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Username:         username,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		Host:             r.Host,
		TotalCommits:     totalCommits,
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if strings.Contains(r.Host, ":") || conf.Server.SSHPort != "22" {
		host := strings.Split(r.Host, ":")[0]
		port := conf.Server.SSHPort
		data.SSHClone = fmt.Sprintf("ssh://%s:%s/%s.git", host, port, reponame)
	} else {
		data.SSHClone = fmt.Sprintf("git@%s:%s.git", r.Host, reponame)
	}

	if totalCommits == "" {
		writeRepoResponse(w, r, db, reponame, "repo-summary-empty.html", data)
		return
	}

	data.RepoDetail.Readme = processREADME(repoDir)

	commits := getCommits(repoDir, "master", -3)
	data.RepoLogs = *commits

	_, totalTags := util.GetGitTags(repoDir)
	data.TotalRefs = totalTags

	contributors := getContributors(repoDir, false)
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

// GetRepoMeta ...
func GetRepoMeta(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	var token string
	if userPresent == "true" {
		token = w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	username := model.GetUsernameFromToken(db, token)
	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)
	repoID := model.GetRepoIDFromReponame(db, reponame)

	grms := model.GetRepoMembers(db, repoID)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Username:         username,
		Reponame:         reponame,
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoMembers:      grms,
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	writeRepoResponse(w, r, db, reponame, "repo-meta.html", data)
	return
}

// PostRepoMetaDelete ...
func PostRepoMetaDelete(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	userPresent := w.Header().Get("user-present")
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		// username := model.GetUsernameFromToken(db, token)

		if model.CheckRepoOwnerFromUserIDAndReponame(db, userID, reponame) {
			model.DeleteRepobyReponame(db, reponame)
			refsPattern := filepath.Join(conf.Paths.RefsPath, reponame+"*")

			files, err := filepath.Glob(refsPattern)
			errorhandler.CheckError("Error on post repo meta delete filepath.Glob", err)

			for _, f := range files {
				err := os.Remove(f)
				errorhandler.CheckError("Error on removing ref files", err)
			}

			repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")
			err = os.RemoveAll(repoDir)
			errorhandler.CheckError("Error on removing repository directory", err)
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// PostRepoMetaStruct struct
type PostRepoMetaStruct struct {
	Name        string `schema:"name"`
	Description string `schema:"description"`
	IsPrivate   string `schema:"is_private"`
}

// PostRepoMeta ...
func PostRepoMeta(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)

		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &errorhandler.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			errorhandler.CheckError("Error on post create repo json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		var postRepoMetaStruct = &PostRepoMetaStruct{}
		err := decoder.Decode(postRepoMetaStruct, r.PostForm)
		errorhandler.CheckError("Error on post repo meta decoder", err)

		s := postRepoMetaStruct.Name
		if len(s) > 100 || len(s) < 1 {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			repoMetaPage := path.Join("./templates", "repo-meta.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, repoMetaPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := GetRepoResponse{
				SiteSettings:       util.GetSiteSettings(db, conf),
				IsLoggedIn:         checkUserLoggedIn(w),
				ShowLoginMenu:      true,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				Username:           username,
				Reponame:           reponame,
				ReponameErrMessage: "Repository name is too long (maximum is 100 characters).",
				RepoDescription:    model.GetRepoDescriptionFromRepoName(db, reponame),
				IsRepoPrivate:      model.GetRepoType(db, reponame),
				RepoAccess:         model.CheckRepoOwnerFromUserIDAndReponame(db, userID, reponame),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		} else if strings.HasPrefix(s, "-") || strings.Contains(s, "--") || strings.HasSuffix(s, "-") || !util.IsAlnumOrHyphen(s) {
			layoutPage := path.Join("./templates", "layout.html")
			headerPage := path.Join("./templates", "header.html")
			repoMetaPage := path.Join("./templates", "repo-meta.html")
			footerPage := path.Join("./templates", "footer.html")

			tmpl, err := template.ParseFiles(layoutPage, headerPage, repoMetaPage, footerPage)
			errorhandler.CheckError("Error on template parse", err)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)

			data := GetRepoResponse{
				SiteSettings:       util.GetSiteSettings(db, conf),
				IsLoggedIn:         checkUserLoggedIn(w),
				ShowLoginMenu:      true,
				HeaderActiveMenu:   "",
				SorciaVersion:      conf.Version,
				Username:           username,
				Reponame:           reponame,
				ReponameErrMessage: "Repository name may only contain alphanumeric characters or single hyphens, and cannot begin or end with a hyphen.",
				RepoDescription:    model.GetRepoDescriptionFromRepoName(db, reponame),
				IsRepoPrivate:      model.GetRepoType(db, reponame),
				RepoAccess:         model.CheckRepoOwnerFromUserIDAndReponame(db, userID, reponame),
			}

			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}

		var isPrivate int
		if isPrivate = 0; postRepoMetaStruct.IsPrivate == "1" {
			isPrivate = 1
		}

		if model.CheckRepoOwnerFromUserIDAndReponame(db, userID, reponame) == true {
			urs := model.UpdateRepoStruct{
				RepoID:      model.GetRepoIDFromReponame(db, reponame),
				NewName:     postRepoMetaStruct.Name,
				Description: postRepoMetaStruct.Description,
				IsPrivate:   isPrivate,
			}

			model.UpdateRepo(db, urs)

			// Update repository dir name
			oldRepoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")
			newRepoDir := filepath.Join(conf.Paths.RepoPath, urs.NewName+".git")

			if _, err := os.Stat(newRepoDir); os.IsNotExist(err) {
				err = os.Rename(oldRepoDir, newRepoDir)
				errorhandler.CheckError("Error on update repository dir name", err)
			}

			util.UpdateRefsWithNewName(conf.Paths.RefsPath, conf.Paths.RepoPath, reponame, urs.NewName)

			http.Redirect(w, r, "/r/"+urs.NewName+"/meta", http.StatusFound)
			return
		}
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

// PostRepoMetaMember struct
type PostRepoMetaMember struct {
	Username   string `schema:"username"`
	Permission string `schema:"is_readorwrite"`
}

// PostRepoMetaUser ...
func PostRepoMetaUser(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct, decoder *schema.Decoder) {
	userPresent := w.Header().Get("user-present")
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		username := model.GetUsernameFromToken(db, token)

		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			errorResponse := &errorhandler.Response{
				Error: err.Error(),
			}

			errorJSON, err := json.Marshal(errorResponse)
			errorhandler.CheckError("Error on post create repo json marshal", err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			w.Write(errorJSON)
		}

		layoutPage := path.Join("./templates", "layout.html")
		headerPage := path.Join("./templates", "header.html")
		repoMetaPage := path.Join("./templates", "repo-meta.html")
		footerPage := path.Join("./templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, repoMetaPage, footerPage)
		errorhandler.CheckError("Error on template parse", err)

		data := GetRepoResponse{
			SiteSettings:     util.GetSiteSettings(db, conf),
			IsLoggedIn:       checkUserLoggedIn(w),
			ShowLoginMenu:    true,
			HeaderActiveMenu: "",
			SorciaVersion:    conf.Version,
			Username:         username,
			Reponame:         reponame,
			RepoDescription:  model.GetRepoDescriptionFromRepoName(db, reponame),
			IsRepoPrivate:    model.GetRepoType(db, reponame),
			RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, model.GetUserIDFromUsername(db, username), reponame),
		}

		var postRepoMetaMember = &PostRepoMetaMember{}
		err = decoder.Decode(postRepoMetaMember, r.PostForm)
		errorhandler.CheckError("Error on post repo meta member decoder", err)

		userID := model.GetUserIDFromUsername(db, postRepoMetaMember.Username)
		repoID := model.GetRepoIDFromReponame(db, reponame)
		if userID > 0 {
			if !model.CheckRepoOwnerFromUserIDAndReponame(db, userID, reponame) {
				if !model.CheckRepoMemberExistFromUserIDAndRepoID(db, userID, repoID) {
					crm := model.CreateRepoMember{
						UserID:     userID,
						RepoID:     repoID,
						Permission: postRepoMetaMember.Permission,
					}

					model.InsertRepoMember(db, crm)

					http.Redirect(w, r, "/r/"+reponame+"/meta", http.StatusFound)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				data.RepoUserAddError = "User is already a member of this repository."
				tmpl.ExecuteTemplate(w, "layout", data)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			data.RepoUserAddError = "User is the owner of this repository."
			tmpl.ExecuteTemplate(w, "layout", data)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		data.RepoUserAddError = "User does not exist. Check if the username is correct or ask the server/sys admin to add this user."
		tmpl.ExecuteTemplate(w, "layout", data)
	}

	http.Redirect(w, r, "/login", http.StatusFound)
	return
}

// GetRepoTree ...
func GetRepoTree(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branch := vars["branch"]

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		RepoBranches:     util.GetGitBranches(repoDir),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gitPath := util.GetGitBinPath()

	dirs, files := walkThrough(repoDir, gitPath, branch, ".", 0)

	data.RepoDetail.WalkPath = r.URL.Path
	data.RepoDetail.PathEmpty = true

	data.RepoDetail.RepoDirsDetail, data.RepoDetail.RepoFilesDetail = applyDirsAndFiles(dirs, files, repoDir, ".", branch)

	commit := getCommits(repoDir, branch, -1)
	data.RepoLogs = *commit
	if len(data.RepoLogs.History) == 1 {
		data.RepoLogs.History[0].Message = util.LimitCharLengthInString(data.RepoLogs.History[0].Message)
	}

	writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
	return
}

// GetRepoTreePath ...
func GetRepoTreePath(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branchOrHash := vars["branchorhash"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")
	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoBranch:     true,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		RepoBranches:     util.GetGitBranches(repoDir),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gitPath := util.GetGitBinPath()
	frdpath := strings.Split(r.URL.Path, "r/"+reponame+"/tree/"+branchOrHash+"/")[1]

	args := []string{"branch"}
	out := util.ForkExec(gitPath, args, repoDir)

	ss := strings.Split(out, "\n")
	entries := ss[:len(ss)-1]

	legendHref := "\"/r/" + reponame + "/tree/" + branchOrHash + "\""
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

	for _, entry := range entries {
		entryCheck := strings.TrimSpace(entry)

		if entryCheck == branchOrHash || entryCheck == fmt.Sprintf("* %s", branchOrHash) {
			frdPathLen := len(strings.Split(frdpath, "/"))
			dirs, files := walkThrough(repoDir, gitPath, branchOrHash, frdpath, frdPathLen)

			if len(dirs) == 0 && len(files) == 0 {
				args := []string{"show", fmt.Sprintf("%s:%s", branchOrHash, frdpath)}
				out := util.ForkExec(gitPath, args, repoDir)

				frdSplit := strings.Split(frdpath, "/")

				frdFile := frdSplit[len(frdSplit)-1]

				fileDotSplit := strings.Split(frdFile, ".")
				var fileContent string
				if len(fileDotSplit) > 1 {
					fileContent = fmt.Sprintf("<pre><code class=\"%s\">%s</code></pre>", fileDotSplit[1], template.HTMLEscaper(out))
				} else {
					fileContent = fmt.Sprintf("<pre><code class=\"plaintext\">%s</code></pre>", template.HTMLEscaper(out))
				}

				data.RepoDetail.FileContent = template.HTML(fileContent)

				data.SiteStyle = model.GetSiteStyle(db)

				writeRepoResponse(w, r, db, reponame, "file-viewer.html", data)
				return
			}

			data.RepoDetail.RepoDirsDetail, data.RepoDetail.RepoFilesDetail = applyDirsAndFiles(dirs, files, repoDir, frdpath, branchOrHash)

			writeRepoResponse(w, r, db, reponame, "repo-tree.html", data)
			return
		}
	}

	data.IsRepoBranch = false

	args = []string{"show", branchOrHash, "--pretty=format:", "--", frdpath}
	out = util.ForkExec(gitPath, args, repoDir)

	diffLine := strings.Split(out, "\n")[0]

	if diffLine == fmt.Sprintf("diff --git a/%s b/%s", frdpath, frdpath) {
		args = []string{"show", fmt.Sprintf("%s:%s", branchOrHash, frdpath)}
		out = util.ForkExec(gitPath, args, repoDir)
	}

	frdSplit := strings.Split(frdpath, "/")

	frdFile := frdSplit[len(frdSplit)-1]

	fileDotSplit := strings.Split(frdFile, ".")
	var fileContent string
	if len(fileDotSplit) > 1 {
		fileContent = fmt.Sprintf("<pre><code class=\"%s\">%s</code></pre>", fileDotSplit[1], template.HTMLEscaper(out))
	} else {
		fileContent = fmt.Sprintf("<pre><code class=\"plaintext\">%s</code></pre>", template.HTMLEscaper(out))
	}

	data.RepoDetail.FileContent = template.HTML(fileContent)

	data.SiteStyle = model.GetSiteStyle(db)

	writeRepoResponse(w, r, db, reponame, "file-viewer.html", data)
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
		} else if lsTreePathLen == 0 && !util.ContainsValueInArr(dirs, entrySplit[0]) {
			dirs = append(dirs, entrySplit[0])
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

// applyDirsAndFiles ...
func applyDirsAndFiles(dirs, files []string, repoDir, frdpath, branch string) ([]RepoDirDetail, []RepoFileDetail) {
	gitPath := util.GetGitBinPath()
	repoDetail := RepoDetail{}

	for _, dir := range dirs {
		dirPath := fmt.Sprintf("%s/%s", frdpath, dir)
		repoDirDetail := RepoDirDetail{}

		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr||srca-sptra||%H", "--", dirPath}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoDirDetail.DirName = dir
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoDirDetail.DirCommit = commit
		repoDirDetail.DirCommitDate = ss[1]
		repoDirDetail.DirCommitFullHash = ss[2]
		repoDirDetail.DirCommitBranch = branch
		repoDetail.RepoDirsDetail = append(repoDetail.RepoDirsDetail, repoDirDetail)
	}

	for _, file := range files {
		filePath := fmt.Sprintf("%s/%s", frdpath, file)
		repoFileDetail := RepoFileDetail{}

		args := []string{"log", branch, "-n", "1", "--pretty=format:%s||srca-sptra||%cr||srca-sptra||%H", "--", filePath}
		out := util.ForkExec(gitPath, args, repoDir)

		ss := strings.Split(out, "||srca-sptra||")

		repoFileDetail.FileName = file
		commit := ss[0]
		if len(commit) > 50 {
			commit = util.LimitCharLengthInString(commit)
		}

		repoFileDetail.FileCommit = commit
		repoFileDetail.FileCommitDate = ss[1]
		repoFileDetail.FileCommitFullHash = ss[2]
		repoFileDetail.FileCommitBranch = branch
		repoDetail.RepoFilesDetail = append(repoDetail.RepoFilesDetail, repoFileDetail)
	}

	return repoDetail.RepoDirsDetail, repoDetail.RepoFilesDetail
}

// GetRepoLog ...
func GetRepoLog(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	branch := vars["branch"]

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

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

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
		RepoBranches:     util.GetGitBranches(repoDir),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	commits := getCommitsFromHash(repoDir, branch, fromHash, 11)
	data.RepoLogs = *commits

	writeRepoResponse(w, r, db, reponame, "repo-log.html", data)
	return
}

// Refs struct
type Refs struct {
	Version   string
	Targz     string
	TargzPath string
	Zip       string
	ZipPath   string
	Message   string
}

// GetRepoRefs ...
func GetRepoRefs(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

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
		tarRefPath := filepath.Join(conf.Paths.RefsPath, tarFilename)

		if _, err := os.Stat(tarRefPath); !os.IsNotExist(err) {
			rf.Targz = tarFilename
			rf.TargzPath = fmt.Sprintf("/dl/%s", tarFilename)
		}

		// Generate zip file
		zipFilename := fmt.Sprintf("%s-%s.zip", reponame, tagname)
		zipRefPath := filepath.Join(conf.Paths.RefsPath, zipFilename)

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

// ServeRefFile ...
func ServeRefFile(w http.ResponseWriter, r *http.Request, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	fileName := vars["file"]
	dlPath := filepath.Join(conf.Paths.RefsPath, fileName)
	http.ServeFile(w, r, dlPath)
}

// GetRepoContributors ...
func GetRepoContributors(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	contributors := getContributors(repoDir, true)

	data.Contributors = *contributors

	writeRepoResponse(w, r, db, reponame, "repo-contributors.html", data)
	return
}

// CommitDetailStruct struct
type CommitDetailStruct struct {
	DP           string
	Name         string
	Message      string
	Hash         string
	Branch       string
	Date         string
	CommitStatus string
	Files        []CommitFile
}

// CommitFile struct
type CommitFile struct {
	Filename     string
	State        string
	PreviousHash string
	Ampersand    string
	CodeLines    template.HTML
}

// GetCommitDetail ...
func GetCommitDetail(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *setting.BaseStruct) {
	vars := mux.Vars(r)
	reponame := vars["reponame"]
	commitHash := vars["hash"]
	branch := vars["branch"]

	repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	userPresent := w.Header().Get("user-present")
	var loggedInUserID int
	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		loggedInUserID = model.GetUserIDFromToken(db, token)
	}

	repoDescription := model.GetRepoDescriptionFromRepoName(db, reponame)

	data := GetRepoResponse{
		SiteSettings:     util.GetSiteSettings(db, conf),
		IsLoggedIn:       checkUserLoggedIn(w),
		ShowLoginMenu:    true,
		HeaderActiveMenu: "",
		SorciaVersion:    conf.Version,
		Reponame:         reponame,
		RepoAccess:       model.CheckRepoOwnerFromUserIDAndReponame(db, loggedInUserID, reponame),
		RepoDescription:  repoDescription,
		IsRepoPrivate:    model.GetRepoType(db, reponame),
	}

	if !data.IsLoggedIn && data.IsRepoPrivate {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	gitPath := util.GetGitBinPath()

	args := []string{"show", commitHash, "--name-status", "--pretty=format:%ae||srca-sptra||%an||srca-sptra||%s||srca-sptra||%ar"}
	out := util.ForkExec(gitPath, args, repoDir)

	lines := strings.Split(out, "\n")
	// Remove empty last line
	lines = lines[:len(lines)-1]

	//filesChanged := strings.TrimSpace(lines[len(lines)-1])
	ss := strings.Split(lines[0], "||srca-sptra||")
	var cds CommitDetailStruct
	cds.Branch = branch

	if len(ss) > 1 {
		cds.Hash = commitHash

		email := ss[0]
		gravatarHash := md5.Sum([]byte(email))
		stringHash := hex.EncodeToString(gravatarHash[:])
		cds.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

		cds.Name = ss[1]
		cds.Message = ss[2]
		cds.Date = ss[3]
	}

	var cf CommitFile
	for _, file := range lines[1:] {
		cf.State = strings.Fields(file)[0]
		cf.Filename = strings.Fields(file)[1]

		args := []string{"show", commitHash, commitHash, "--pretty=format:", "--full-index", "--", cf.Filename}
		out := util.ForkExec(gitPath, args, repoDir)

		lines = strings.Split(out, "\n")
		// Remove empty last line
		lines = lines[:len(lines)-1]

		// Get PreviousHash and Ampersand
		for i, line := range lines {
			ts := strings.TrimSpace(line)
			if strings.HasPrefix(ts, fmt.Sprintf("diff --git a/%s b/%s", cf.Filename, cf.Filename)) {
				indexSplit := strings.Fields(strings.TrimSpace(lines[i+1]))
				cf.PreviousHash = strings.Split(indexSplit[1], "..")[0]
			}

			if strings.HasPrefix(ts, "@@") {
				cf.Ampersand = ts
			}
		}

		line := strings.Join(lines[5:], "\n")
		cf.CodeLines = template.HTML(template.HTMLEscaper(line))

		cds.Files = append(cds.Files, cf)
	}

	// Get commit status
	args = []string{"show", commitHash, "--stat", "--pretty=format:"}
	out = util.ForkExec(gitPath, args, repoDir)

	lines = strings.Split(out, "\n")
	// Remove empty last line
	lines = lines[:len(lines)-1]
	commitStatus := strings.TrimSpace(lines[len(lines)-1])
	cds.CommitStatus = commitStatus

	data.CommitDetail = cds

	writeRepoResponse(w, r, db, reponame, "repo-commit.html", data)
	return
}

// Contributors struct
type Contributors struct {
	Detail []Contributor
	Total  string
}

// Contributor struct
type Contributor struct {
	Name    string
	DP      string
	Commits string
}

func getContributors(repoDir string, getDetail bool) *Contributors {
	gitPath := util.GetGitBinPath()

	args := []string{"shortlog", "HEAD", "-sne"}
	out := util.ForkExec(gitPath, args, repoDir)

	cStringRmLastLn := strings.TrimSuffix(out, "\n")
	lines := strings.Split(cStringRmLastLn, "\n")

	var contributors Contributors

	contributors.Total = strconv.Itoa(len(lines))

	if getDetail {
		for _, line := range lines {
			lineDetail := strings.Fields(line)
			var contributor Contributor
			if len(lineDetail) > 1 {
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
	}

	return &contributors
}

func noRepoAccess(w http.ResponseWriter) {
	errorResponse := &errorhandler.Response{
		Error: "You don't have access to this repository.",
	}

	errorJSON, err := json.Marshal(errorResponse)
	errorhandler.CheckError("Error on no repo access function json marshal", err)

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
			if hasRepoAccess := model.CheckRepoOwnerFromUserIDAndReponame(db, userIDFromToken, reponame); hasRepoAccess {
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
	errorhandler.CheckError("Error on template parse", err)

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

func getCommits(repoDir, branch string, commitCount int) *RepoLogs {
	rla := RepoLogs{}
	rl := RepoLog{}

	gitPath := util.GetGitBinPath()

	var args []string
	args = []string{"log", branch, strconv.Itoa(commitCount), "--pretty=format:%H||srca-sptra||%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae"}
	out := util.ForkExec(gitPath, args, repoDir)

	ss := strings.Split(out, "\n")

	for i := 0; i < len(ss); i++ {
		st := strings.Split(ss[i], "||srca-sptra||")
		if len(st) > 1 {
			rl.FullHash = st[0]
			rl.Hash = st[1]
			rl.Message = st[3]
			rl.Date = st[4]
			rl.Author = st[5]
			rl.Branch = branch

			hash := md5.Sum([]byte(st[6]))
			stringHash := hex.EncodeToString(hash[:])
			rl.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

			rla = RepoLogs{
				History: append(rla.History, rl),
			}
		}
	}

	return &rla
}

func getCommitsFromHash(repoDir, branch, fromHash string, commitCount int) *RepoLogs {
	rla := RepoLogs{}
	rl := RepoLog{}

	var hashLink string

	ss := getGitCommits(commitCount, branch, fromHash, repoDir)

	for i := 0; i < len(ss); i++ {
		if i == (len(ss) - 1) {
			hashLink = strings.Split(ss[i], "||srca-sptra||")[0]

			gitPath := util.GetGitBinPath()
			args := []string{"rev-list", branch, "--max-parents=0", "HEAD"}
			out := util.ForkExec(gitPath, args, repoDir)

			lastHash := strings.Split(out, "\n")[0]

			if hashLink != lastHash {
				rla.IsNext = true
				break
			}
		}
		st := strings.Split(ss[i], "||srca-sptra||")
		if len(st) > 1 {
			rl.FullHash = st[0]
			rl.Hash = st[1]
			rl.Message = st[3]
			rl.Date = st[4]
			rl.Author = st[5]
			rl.Branch = branch

			hash := md5.Sum([]byte(st[6]))
			stringHash := hex.EncodeToString(hash[:])
			rl.DP = fmt.Sprintf("https://www.gravatar.com/avatar/%s", stringHash)

			rla = RepoLogs{
				History: append(rla.History, rl),
			}
		}
	}

	rla.HashLink = hashLink

	return &rla
}

func getGitCommits(commitCount int, branch, fromHash, dirPath string) []string {
	gitPath := util.GetGitBinPath()

	var args []string
	if fromHash == "" {
		args = []string{"log", branch, fmt.Sprintf("--max-count=%s", strconv.Itoa(commitCount)), "--pretty=format:%H||srca-sptra||%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae"}
	} else {
		args = []string{"log", fmt.Sprintf("--max-count=%s", strconv.Itoa(commitCount)), fromHash, "--pretty=format:%H||srca-sptra||%h||srca-sptra||%d||srca-sptra||%s||srca-sptra||%cr||srca-sptra||%an||srca-sptra||%ae"}
	}
	out := util.ForkExec(gitPath, args, dirPath)

	ss := strings.Split(out, "\n")

	return ss
}
