package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"path"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gin-gonic/gin"
)

// GetCreateRepoResponse struct
type GetCreateRepoResponse struct {
	Username string
}

// GetCreateRepo ...
func GetCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := r.Header.Get("sorcia-token")
		username := model.GetUsernameFromToken(db, token)

		tmpl := template.Must(template.ParseFiles("./templates/create-repo.html"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := GetCreateRepoResponse{
			Username: username,
		}

		tmpl.Execute(w, data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `form:"name" binding:"required"`
	Description string `form:"description" binding:"required"`
	IsPrivate   string `form:"is_private" binding:"required"`
}

// PostCreateRepo ...
func PostCreateRepo(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	createRepoRequest := &CreateRepoRequest{
		Name:        r.FormValue("username"),
		Description: r.FormValue("description"),
		IsPrivate:   r.FormValue("is_private"),
	}

	token := r.Header.Get("sorcia-token")

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

	conf := setting.GetConf()

	// Create Git bare repository
	bareRepoDir := path.Join(conf.Paths.DataPath, "repositories/"+"+"+username+"/"+createRepoRequest.Name+".git")

	cmd := exec.Command("git", "init", "--bare", bareRepoDir)
	err := cmd.Run()
	errorhandler.CheckError(err)

	// Clone from the bare repository created above
	repoDir := path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+createRepoRequest.Name)
	cmd = exec.Command("git", "clone", bareRepoDir, repoDir)
	err = cmd.Run()
	errorhandler.CheckError(err)

	http.Redirect(w, r, "/", http.StatusFound)
}

// GetRepo ...
func GetRepo(c *gin.Context) {
	username := c.Param("username")
	reponame := c.Param("reponame")

	db, ok := c.MustGet("db").(*sql.DB)
	if !ok {
		fmt.Println("Middleware db error")
	}

	rts := model.RepoTypeStruct{
		Username: username,
		Reponame: reponame,
	}

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		c.HTML(http.StatusNotFound, "", "")
		return
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		c.HTML(http.StatusOK, "repo-summary.html", gin.H{
			"username": rts.Username,
			"reponame": rts.Reponame,
		})
	} else {
		userPresent, ok := c.MustGet("userPresent").(bool)
		if !ok {
			fmt.Println("Middleware user error")
		}

		if userPresent {
			token, _ := c.Cookie("sorcia-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				c.HTML(http.StatusOK, "repo-summary.html", gin.H{
					"username": rts.Username,
					"reponame": rts.Reponame,
				})
			} else {
				c.HTML(http.StatusNotFound, "", "")
			}
		} else {
			c.Redirect(http.StatusMovedPermanently, "/")
		}
	}
}

// GetRepoTree ...
func GetRepoTree(c *gin.Context) {
	username := c.Param("username")
	reponame := c.Param("reponame")

	db, ok := c.MustGet("db").(*sql.DB)
	if !ok {
		fmt.Println("Middleware db error")
	}

	rts := model.RepoTypeStruct{
		Username: username,
		Reponame: reponame,
	}

	if repoExists := model.CheckRepoExists(db, reponame); !repoExists {
		c.HTML(http.StatusNotFound, "", "")
		return
	}

	// Check if repository is not private
	if isRepoPrivate := model.GetRepoType(db, &rts); !isRepoPrivate {
		c.HTML(http.StatusOK, "repo-tree.html", gin.H{
			"username": rts.Username,
			"reponame": rts.Reponame,
		})
	} else {
		userPresent, ok := c.MustGet("userPresent").(bool)
		if !ok {
			fmt.Println("Middleware user error")
		}

		if userPresent {
			token, _ := c.Cookie("sorcia-token")
			userIDFromToken := model.GetUserIDFromToken(db, token)

			// Check if the logged in user has access to view the repository.
			if hasRepoAccess := model.CheckRepoAccessFromUserID(db, userIDFromToken); hasRepoAccess {
				c.HTML(http.StatusOK, "repo-tree.html", gin.H{
					"username": rts.Username,
					"reponame": rts.Reponame,
				})
			} else {
				c.HTML(http.StatusNotFound, "", "")
			}
		} else {
			c.Redirect(http.StatusMovedPermanently, "/")
		}
	}
}
