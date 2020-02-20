package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"

	errorhandler "sorcia/error"
	"sorcia/model"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
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
		layoutPage := path.Join("./templates", "layout.tmpl")
		headerPage := path.Join("./templates", "header.tmpl")
		createRepoPage := path.Join("./templates", "create-repo.tmpl")
		footerPage := path.Join("./templates", "footer.tmpl")

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
	Host             string
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
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
		Host:             r.Host,
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		layoutPage := path.Join("./templates", "layout.tmpl")
		headerPage := path.Join("./templates", "header.tmpl")
		repoSummaryPage := path.Join("./templates", "repo-summary.tmpl")
		footerPage := path.Join("./templates", "footer.tmpl")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, repoSummaryPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		userPresent := w.Header().Get("user-present")

		if userPresent == "true" {
			token := w.Header().Get("sorcia-cookie-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				layoutPage := path.Join("./templates", "layout.tmpl")
				headerPage := path.Join("./templates", "header.tmpl")
				repoSummaryPage := path.Join("./templates", "repo-summary.tmpl")
				footerPage := path.Join("./templates", "footer.tmpl")

				tmpl, err := template.ParseFiles(layoutPage, headerPage, repoSummaryPage, footerPage)
				errorhandler.CheckError(err)

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)

				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

// GetRepoTree ...
func GetRepoTree(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	vars := mux.Vars(r)
	username := vars["username"]
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
		Username:         username,
		Reponame:         reponame,
	}

	layoutPage := path.Join("./templates", "layout.tmpl")
	headerPage := path.Join("./templates", "header.tmpl")
	repoTreePage := path.Join("./templates", "repo-tree.tmpl")
	footerPage := path.Join("./templates", "footer.tmpl")

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
				tmpl.ExecuteTemplate(w, "layout", data)
			} else {
				noRepoAccess(w)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
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
