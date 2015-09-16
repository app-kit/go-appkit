package utils

import (
	"os"
	"path"
	"fmt"
	"io/ioutil"

	"github.com/twinj/uuid"

	. "github.com/theduke/go-appkit/error"
)

func AbsPath(p string) (string, Error) {
	if !path.IsAbs(p) { 
		wd, err := os.Getwd() 
		if err != nil { 
	  	return "", AppError{
	  		Code: "get_wd_error",
	  		Message: err.Error(),
	  		Internal: true,
	  	}
		}
		p = path.Clean(path.Join(wd, p))
	}

	return p, nil
}

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

func WriteFile(p string, content []byte, createDir bool) Error {
	if createDir {
		dir, err := AbsPath(path.Dir(p))
		if err != nil {
			return err
		}

		if dir != "" {
			if err := os.MkdirAll(dir, 0777); err != nil {
				return AppError{
					Code: "mkdir_error",
					Message: err.Error(),
					Internal: true,
				}
			}
		}
	}

	f, err := os.Create(p)
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

// Write contents to a tmp file and return the path to the file.
func WriteTmpFile(content[]byte, name string) (string, Error) {
	if name == "" {
		name = uuid.NewV4().String()
	} else if name[0] == '.' {
		name = uuid.NewV4().String() + name
	}

	p := path.Join(os.TempDir(), "tmpfiles", name)
	if err := WriteFile(p, content, true); err != nil {
		return "", err
	}

	return p, nil
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
