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

// UserMod ...
func UserMod(conf *setting.BaseStruct) {
	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Reset user password\n[2] Delete user")
		fmt.Print("Enter your option [1/2]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError("User mod", err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			resetUserPassword(db)

			return
		case "2":
			deleteUser(db, conf)

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
		errorhandler.CheckError("Reset user password", err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			fmt.Print("Enter username: ")
			usernameInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Error on Username", err)

			fmt.Print("Enter new password: ")
			passwordInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Error on Password", err)

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
			errorhandler.CheckError("Error on email address: ", err)

			fmt.Print("Enter new password: ")
			passwordInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Error on password", err)

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
	errorhandler.CheckError("Error on generating password hash", err)

	// Generate JWT token using the hash password above
	token, err := handler.GenerateJWTToken(passwordHash)
	errorhandler.CheckError("Error on generating jwt token", err)

	return passwordHash, token
}

func deleteUser(db *sql.DB, conf *setting.BaseStruct) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("[1] Specify user by username\n[2] Specify user by email address")
		fmt.Print("Enter your option [1/2]: ")
		optionInput, err := reader.ReadString('\n')
		errorhandler.CheckError("Delete user", err)

		option := strings.TrimSpace(optionInput)

		switch option {
		case "1":
			fmt.Print("Enter username: ")
			usernameInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Error on username", err)

			username := strings.TrimSpace(usernameInput)
			model.DeleteUserbyUsername(db, username)

			return
		case "2":
			fmt.Print("Enter email address: ")
			emailInput, err := reader.ReadString('\n')
			errorhandler.CheckError("Error on email address", err)

			email := strings.TrimSpace(emailInput)
			model.DeleteUserbyEmail(db, email)

			return
		default:
			fmt.Println("Unknown option - expected 1 or 2")
		}
	}
}
