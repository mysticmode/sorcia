package error

import (
	"log"
)

// CheckError ...
func CheckError(errMessage string, err error) {
	if err != nil {
		log.Printf("%s: %v", errMessage, err)
	}
}

// Response struct
type Response struct {
	Error string `json:"error"`
}
