package error

import (
	"io"
	"log"
	"os"
)

// CheckError ...
func CheckError(errMessage string, err error) {
	if err != nil {
		f, err := os.OpenFile("./error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		defer f.Close()
		wrt := io.MultiWriter(os.Stdout, f)
		log.SetOutput(wrt)
		log.Printf("%s: %v", errMessage, err)
	}
}

// Response struct
type Response struct {
	Error string `json:"error"`
}
