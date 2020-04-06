package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"sorcia/internal"
	"sorcia/models"
	"sorcia/pkg"
)

// UserMod ...
func UserMod(conf *pkg.BaseStruct) {
	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Reset username of a user\n[2] Reset password of a user\n[3] Delete user\n[4] Delete repository")
		fmt.Print("Enter your option [1/2/3/4]: ")
		optionInput, err := reader.ReadString('\n')
		pkg.CheckError("User mod", err)

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
		pkg.CheckError("Usermod: Reset username of a user", err)

		username := strings.TrimSpace(usernameInput)

		userID := models.GetUserIDFromUsername(db, username)

		if userID > 0 {
			fmt.Println("Enter the new username")
			newUsernameInput, err := reader.ReadString('\n')
			pkg.CheckError("Usermod: new username error", err)

			newUsername := strings.TrimSpace(newUsernameInput)

			models.ResetUsernameByUserID(db, newUsername, userID)
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
		pkg.CheckError("Usermod: Reset password of a user - enter username", err)

		username := strings.TrimSpace(usernameInput)

		userID := models.GetUserIDFromUsername(db, username)

		if userID > 0 {
			newPassword := getPassword("Enter the password")

			// Generate password hash using bcrypt
			passwordHash, err := internal.HashPassword(newPassword)
			pkg.CheckError("Error on usermod hash password", err)

			// Generate JWT token using the hash password above
			token, err := internal.GenerateJWTToken(passwordHash)
			pkg.CheckError("Error on usermod generate jwt token", err)

			rsp := models.ResetUserPasswordbyUsernameStruct{
				Username:     username,
				PasswordHash: passwordHash,
				JwtToken:     token,
			}

			models.ResetUserPasswordbyUsername(db, rsp)

			fmt.Println("Password has been successfully changed.")
			return
		}
		fmt.Println("Username does not exist. Please check the username or Ctrl-c to exit")
	}
}

func getPassword(prompt string) string {
	fmt.Println(prompt)

	// Common internals and variables for both stty calls.
	attrs := syscall.ProcAttr{
		Dir:   "",
		Env:   []string{},
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		Sys:   nil}
	var ws syscall.WaitStatus

	// Disable echoing.
	pid, err := syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "-echo"},
		&attrs)
	if err != nil {
		panic(err)
	}

	// Wait for the stty process to complete.
	_, err = syscall.Wait4(pid, &ws, 0, nil)
	if err != nil {
		panic(err)
	}

	// Echo is disabled, now grab the data.
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	// Re-enable echo.
	pid, err = syscall.ForkExec(
		"/bin/stty",
		[]string{"stty", "echo"},
		&attrs)
	if err != nil {
		panic(err)
	}

	// Wait for the stty process to complete.
	_, err = syscall.Wait4(pid, &ws, 0, nil)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(text)
}

func deleteUser(db *sql.DB, conf *pkg.BaseStruct) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the username (this will delete all the repositories that this user has ownership of)")
		usernameInput, err := reader.ReadString('\n')
		pkg.CheckError("Usermod: Reset username of a user", err)

		username := strings.TrimSpace(usernameInput)

		userID := models.GetUserIDFromUsername(db, username)

		if userID > 0 {

			isAdmin := models.CheckifUserIsAnAdmin(db, userID)

			if isAdmin {
				fmt.Println("You cannot delete an admin user of Sorcia.")
				return
			}

			rds := models.GetReposFromUserID(db, userID)

			for _, repo := range rds.Repositories {
				refsPattern := filepath.Join(conf.Paths.RefsPath, repo.Name+"*")
				files, err := filepath.Glob(refsPattern)
				pkg.CheckError("Error on post repo meta delete filepath.Glob", err)

				for _, f := range files {
					err := os.Remove(f)
					pkg.CheckError("Error on removing ref files", err)
				}

				repoDir := filepath.Join(conf.Paths.RepoPath, repo.Name+".git")
				err = os.RemoveAll(repoDir)
				pkg.CheckError("Error on removing repository directory", err)
			}

			rmi := models.GetRepoMemberIDFromUserID(db, userID)

			for _, m := range rmi {
				models.DeleteRepoMemberByID(db, m)
			}

			models.DeleteUserbyUsername(db, username)

			fmt.Println("Username has been successfully deleted.")
			return
		}
		fmt.Println("Username does not exist. Please check the username or Ctrl-c to exit")
	}
}

func deleteRepository(db *sql.DB, conf *pkg.BaseStruct) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter the repository name")
		reponameInput, err := reader.ReadString('\n')
		pkg.CheckError("Usermod: Delete repository", err)

		reponame := strings.TrimSpace(reponameInput)

		if models.CheckRepoExists(db, reponame) {
			models.DeleteRepobyReponame(db, reponame)
			refsPattern := filepath.Join(conf.Paths.RefsPath, reponame+"*")

			files, err := filepath.Glob(refsPattern)
			pkg.CheckError("Error on post repo meta delete filepath.Glob", err)

			for _, f := range files {
				err := os.Remove(f)
				pkg.CheckError("Error on removing ref files", err)
			}

			repoDir := filepath.Join(conf.Paths.RepoPath, reponame+".git")
			err = os.RemoveAll(repoDir)
			pkg.CheckError("Error on removing repository directory", err)

			fmt.Println("Repository has been successfully deleted.")
			return
		}
		fmt.Println("Repository name does not exist. Please check the name or Ctrl-c to exit")
	}
}
