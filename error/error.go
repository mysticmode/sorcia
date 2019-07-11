package error

// CheckError ...
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

// ErrorResponse struct
type ErrorResponse struct {
	Error string `json:"error"`
}
