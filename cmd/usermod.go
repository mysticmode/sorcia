package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/model"
	"sorcia/setting"
)

// UserMod ...
func UserMod(conf *setting.BaseStruct) {
	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Reset username of a user\n[2] Reset password of a user\n[3] Delete user\n[4] Delete repository")
		fmt.Print("Enter your option [1/2/3/4]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError("User mod", err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			resetUserName(db)
			return
		case "2":
			resetUserPassword(db)
			return
		case "3":
			deleteUser(db, conf)
			return
		case "4":
			deleteRepository(db, conf)
			return
		default:
			fmt.Println("Unknown option - expected 1/2/3/4")
		}
	}
}

func resetUserName(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the current username of a user")
		usernameInput, err := reader.ReadString('\n')
		errorhandler.CheckError("Usermod: Reset username of a user", err)

		username := strings.TrimSpace(usernameInput)

		userID := model.GetUserIDFromUsername(db, username)

		if userID > 0 {
			fmt.Println("Enter the new username")
			newUsernameInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Usermod: new username error", err)

			newUsername := strings.TrimSpace(newUsernameInput)

			model.ResetUsernameByUserID(db, newUsername, userID)
			fmt.Println("Username has been successfully changed.")
			return
		}
		fmt.Println("Username does not exist. Please check the username or Ctrl-c to exit")
	}
}

func resetUserPassword(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the username")
		usernameInput, err := reader.ReadString('\n')
		errorhandler.CheckError("Usermod: Reset password of a user - enter username", err)

		username := strings.TrimSpace(usernameInput)

		userID := model.GetUserIDFromUsername(db, username)

		if userID > 0 {
			fmt.Println("Enter the password")
			newPasswordInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Usermod: reset password error", err)

			newPassword := strings.TrimSpace(newPasswordInput)

			// Generate password hash using bcrypt
			passwordHash, err := handler.HashPassword(newPassword)
			errorhandler.CheckError("Error on usermod hash password", err)

			// Generate JWT token using the hash password above
			token, err := handler.GenerateJWTToken(passwordHash)
			errorhandler.CheckError("Error on usermod generate jwt token", err)

			rsp := model.ResetUserPasswordbyUsernameStruct{
				Username:     username,
				PasswordHash: passwordHash,
				JwtToken:     token,
			}

			model.ResetUserPasswordbyUsername(db, rsp)

			fmt.Println("Password has been successfully changed.")
			return
		}
		fmt.Println("Username does not exist. Please check the username or Ctrl-c to exit")
	}
}

func deleteUser(db *sql.DB, conf *setting.BaseStruct) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the username (this will delete all the repositories that this user has ownership of)")
		usernameInput, err := reader.ReadString('\n')
		errorhandler.CheckError("Usermod: Reset username of a user", err)

		username := strings.TrimSpace(usernameInput)

		userID := model.GetUserIDFromUsername(db, username)

		if userID > 0 {

			rds := model.GetReposFromUserID(db, userID)

			for _, repo := range rds.Repositories {
				refsPattern := filepath.Join(conf.Paths.RefsPath, repo.Name+"*")
				files, err := filepath.Glob(refsPattern)
				errorhandler.CheckError("Error on post repo meta delete filepath.Glob", err)

				for _, f := range files {
					err := os.Remove(f)
					errorhandler.CheckError("Error on removing ref files", err)
				}

				repoDir := filepath.Join(conf.Paths.RepoPath, repo.Name+".git")
				err = os.RemoveAll(repoDir)
				errorhandler.CheckError("Error on removing repository directory", err)
			}

			rmi := model.GetRepoMemberIDFromUserID(db, userID)

			for _, m := range rmi {
				model.DeleteRepoMemberByID(db, m)
			}

			model.DeleteUserbyUsername(db, username)

			fmt.Println("Username has been successfully deleted.")
			return
		}
		fmt.Println("Username does not exist. Please check the username or Ctrl-c to exit")
	}
}

func deleteRepository(db *sql.DB, conf *setting.BaseStruct) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the repository name")
		reponameInput, err := reader.ReadString('\n')
		errorhandler.CheckError("Usermod: Delete repository", err)

		reponame := strings.TrimSpace(reponameInput)

		if model.CheckRepoExists(db, reponame) {
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

			fmt.Println("Repository has been successfully deleted.")
			return
		}
		fmt.Println("Repository name does not exist. Please check the name or Ctrl-c to exit")
	}
}
