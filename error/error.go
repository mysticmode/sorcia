package error

import "fmt"

// CheckError ...
func CheckError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// ErrorResponse struct
type ErrorResponse struct {
	Error string `json:"error"`
}
