package error

import "fmt"

// CheckError ...
func CheckError(err error) {
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
}

// Response struct
type Response struct {
	Error string `json:"error"`
}
