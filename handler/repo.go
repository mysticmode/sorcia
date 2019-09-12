package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"path"

	errorhandler "sorcia/error"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gin-gonic/gin"
)

// GetCreateRepo ...
func GetCreateRepo(c *gin.Context) {
	userPresent, ok := c.MustGet("userPresent").(bool)
	if !ok {
		fmt.Println("Middleware user error")
	}

	if userPresent {
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		token, _ := c.Cookie("sorcia-token")

		username := model.GetUsernameFromToken(db, token)

		c.HTML(http.StatusOK, "create-repo.html", gin.H{
			"username": username,
		})
	} else {
		c.Redirect(http.StatusMovedPermanently, "/login")
	}
}

// CreateRepoRequest struct
type CreateRepoRequest struct {
	Name        string `form:"name" binding:"required"`
	Description string `form:"description" binding:"required"`
	IsPrivate   string `form:"is_private" binding:"required"`
}

// PostCreateRepo ...
func PostCreateRepo(c *gin.Context) {
	var form CreateRepoRequest

	if err := c.Bind(&form); err == nil {
		db, ok := c.MustGet("db").(*sql.DB)
		if !ok {
			fmt.Println("Middleware db error")
		}

		token, _ := c.Cookie("sorcia-token")

		userID := model.GetUserIDFromToken(db, token)

		var isPrivate int
		if isPrivate = 0; form.IsPrivate == "1" {
			isPrivate = 1
		}

		crs := model.CreateRepoStruct{
			Name:        form.Name,
			Description: form.Description,
			IsPrivate:   isPrivate,
			UserID:      userID,
		}

		model.InsertRepo(db, crs)

		// Get config values
		conf := setting.GetConf()

		username := model.GetUsernameFromToken(db, token)

		// Create Git bare repository
		bareRepoDir := path.Join(conf.Paths.DataPath, "repositories/"+"+"+username+"/"+form.Name+".git")

		cmd := exec.Command("git", "init", "--bare", bareRepoDir)
		err := cmd.Run()
		errorhandler.CheckError(err)

		// Clone from the bare repository created above
		repoDir := path.Join(conf.Paths.DataPath, "repositories/"+username+"/"+form.Name)
		cmd = exec.Command("git", "clone", bareRepoDir, repoDir)
		err = cmd.Run()
		errorhandler.CheckError(err)

		c.Redirect(http.StatusMovedPermanently, "/")
	} else {
		errorResponse := &errorhandler.ErrorResponse{
			Error: err.Error(),
		}
		c.JSON(http.StatusBadRequest, errorResponse)
	}
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
