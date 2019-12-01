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
	Username         string
}

// GetCreateRepo ...
func GetCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		username := model.GetUsernameFromToken(db, token)

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
			Username:         username,
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
		errorResponse := &errorhandler.ErrorResponse{
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

	username := model.GetUsernameFromToken(db, token)

	// Create Git bare repository
	bareRepoDir := filepath.Join(repoPath, "repositories", username, createRepoRequest.Name+".git")

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	err = cmd.Run()
	errorhandler.CheckError(err)

	// Clone from the bare repository created above
	repoDir := filepath.Join(repoPath, "repositories", username, createRepoRequest.Name)
	cmd = exec.Command("git", "clone", bareRepoDir, repoDir)
	err = cmd.Run()
	errorhandler.CheckError(err)

	http.Redirect(w, r, "/", http.StatusFound)
}

// GetRepoResponse struct
type GetRepoResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	Username         string
	Reponame         string
}

// GetRepo ...
func GetRepo(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	username := vars["username"]
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rts := model.RepoTypeStruct{
		Username: username,
		Reponame: reponame,
	}

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
		Username:         username,
		Reponame:         reponame,
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
				errorResponse := &errorhandler.ErrorResponse{
					Error: "You don't have access to this repository.",
				}

				errorJSON, err := json.Marshal(errorResponse)
				errorhandler.CheckError(err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				w.Write(errorJSON)
			}
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}
}

// GetRepoTree ...
func GetRepoTree(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	username := vars["username"]
	reponame := vars["reponame"]

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	rts := model.RepoTypeStruct{
		Username: username,
		Reponame: reponame,
	}

	data := GetRepoResponse{
		IsHeaderLogin:    false,
		HeaderActiveMenu: "",
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
	errorResponse := &errorhandler.ErrorResponse{
		Error: "You don't have access to this repository.",
	}

	errorJSON, err := json.Marshal(errorResponse)
	errorhandler.CheckError(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	w.Write(errorJSON)
}
