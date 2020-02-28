package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/model"
	"sorcia/setting"
)

func UserMod(conf *setting.BaseStruct) {
	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Reset user password\n[2] Delete user")
		fmt.Print("Enter your option [1/2]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError(err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			resetUserPassword(db)

			return
		case "2":
			deleteUser(db)

			return
		default:
			fmt.Println("Unknown option - expected 1 or 2")
		}
	}
}

func resetUserPassword(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Specify user by username\n[2] Specify user by email address")
		fmt.Print("Enter your option [1/2]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError(err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			fmt.Print("Enter username: ")
			usernameInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			fmt.Print("Enter new password: ")
			passwordInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			username := strings.TrimSpace(usernameInput)
			password := strings.TrimSpace(passwordInput)

			passwordHash, token := generateHashandToken(password)

			resetPass := model.ResetUserPasswordbyUsernameStruct{
				PasswordHash: passwordHash,
				JwtToken:     token,
				Username:     username,
			}
			model.ResetUserPasswordbyUsername(db, resetPass)

			return
		case "2":
			fmt.Print("Enter email address: ")
			emailInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			fmt.Print("Enter new password: ")
			passwordInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			email := strings.TrimSpace(emailInput)
			password := strings.TrimSpace(passwordInput)

			passwordHash, token := generateHashandToken(password)

			resetPass := model.ResetUserPasswordbyEmailStruct{
				PasswordHash: passwordHash,
				JwtToken:     token,
				Email:        email,
			}
			model.ResetUserPasswordbyEmail(db, resetPass)

			return
		default:
			fmt.Println("Unknown option - expected 1 or 2")
		}
	}
}

func generateHashandToken(password string) (string, string) {
	// Generate password hash using bcrypt
	passwordHash, err := handler.HashPassword(password)
	errorhandler.CheckError(err)

	// Generate JWT token using the hash password above
	token, err := handler.GenerateJWTToken(passwordHash)
	errorhandler.CheckError(err)

	return passwordHash, token
}

func deleteUser(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Specify user by username\n[2] Specify user by email address")
		fmt.Print("Enter your option [1/2]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError(err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			fmt.Print("Enter username: ")
			usernameInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			username := strings.TrimSpace(usernameInput)
			model.DeleteUserbyUsername(db, username)

			return
		case "2":
			fmt.Print("Enter email address: ")
			emailInput, err := reader.ReadString('\n')
			errorhandler.CheckError(err)

			email := strings.TrimSpace(emailInput)
			model.DeleteUserbyEmail(db, email)

			return
		default:
			fmt.Println("Unknown option - expected 1 or 2")
		}
	}
}
