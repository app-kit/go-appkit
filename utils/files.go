package utils

import (
	"os"
	"fmt"
	"io/ioutil"

	. "github.com/theduke/go-appkit/error"
)



func FileExists(path string) (bool, Error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, AppError{
				Code: "file_read_error",
				Message: err.Error(),
				Internal: true,
			}
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

func WriteFile(path string, content []byte) Error {
	f, err := os.Create(path)
	if err != nil {
		return AppError{
			Code: "file_create_error",
			Message: err.Error(),
			Internal: true,
		}
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		return AppError{
			Code: "file_write_error",
			Message: err.Error(),
			Internal: true,
		}
	}

	return nil
}

func ListFiles(path string) ([]string, Error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, AppError{
			Code:    "open_dir_error",
			Message: err.Error(),
			Internal: true,
		}
	}
	defer dir.Close()

	items, err := dir.Readdir(-1)
	if err != nil {
		return nil, AppError{
			Code:    "read_error",
			Message: err.Error(),
		}
	}

	files := make([]string, 0)
	for _, item := range items {
		if !item.IsDir() {
			files = append(files, item.Name())
		}
	}

	return files, nil
}
