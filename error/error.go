package error

import (
	"fmt"
	"os"
)

// CheckError ...
func CheckError(err error) {
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// Response struct
type Response struct {
	Error string `json:"error"`
}
