package error

// CheckError ...
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}
