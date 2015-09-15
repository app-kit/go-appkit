package utils

import (
	"os"
	"fmt"
	"io/ioutil"

	. "github.com/theduke/go-appkit/error"
)



func FileExists(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if err == os.ErrNotExist {
			return false, nil
		} else {
			return false, err
		}
	}
	f.Close()

	return true, nil
}

func ReadFile(path string) ([]byte, Error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, AppError{
			Code: "file_open_error",
			Message: fmt.Sprintf("Could not open file at %v: %v", path, err),
			Errors: []error{err},
			Internal: true,
		}
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, AppError{
			Code: "file_read_error",
			Message: fmt.Sprintf("Could not read file at %v: %v", path, err),
			Errors: []error{err},
			Internal: true,
		}
	}

	return content, nil
}
