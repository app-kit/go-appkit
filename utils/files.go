package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"

	"github.com/theduke/go-apperror"
	"github.com/twinj/uuid"
)

func AbsPath(p string) (string, apperror.Error) {
	if !path.IsAbs(p) {
		wd, err := os.Getwd()
		if err != nil {
			return "", apperror.Wrap(err, "get_wd_error")
		}
		p = path.Clean(path.Join(wd, p))
	}

	return p, nil
}

func FileExists(path string) (bool, apperror.Error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, apperror.Wrap(err, "file_read_error")
		}
	}
	f.Close()

	return true, nil
}

func ReadFile(path string) ([]byte, apperror.Error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, apperror.Wrap(err, "file_open_error",
			fmt.Sprintf("Could not open file at %v", path))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, apperror.Wrap(err, "file_read_error",
			fmt.Sprintf("Could not read file at %v", path))
	}

	return content, nil
}

func WriteFile(p string, content []byte, createDir bool) apperror.Error {
	if createDir {
		dir, err := AbsPath(path.Dir(p))
		if err != nil {
			return err
		}

		if dir != "" {
			if err := os.MkdirAll(dir, 0777); err != nil {
				return apperror.Wrap(err, "mkdir_error")
			}
		}
	}

	f, err := os.Create(p)
	if err != nil {
		return apperror.Wrap(err, "file_create_error")
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		return apperror.Wrap(err, "file_write_error")
	}

	return nil
}

func CopyFile(sourcePath, targetPath string) apperror.Error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return apperror.Wrap(err, "file_open_error",
			fmt.Sprintf("Could not open file at %v", sourcePath))
	}
	defer source.Close()

	target, err := os.Open(targetPath)
	if err != nil {
		return apperror.Wrap(err, "file_open_error",
			fmt.Sprintf("Could not open file at %v", targetPath))
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return apperror.Wrap(err, "file_copy_error", fmt.Sprintf("Could not copy file from %v to %v", sourcePath, targetPath))
	}

	return nil
}

// Write contents to a tmp file and return the path to the file.
func WriteTmpFile(content []byte, name string) (string, apperror.Error) {
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

func ListFiles(path string) ([]string, apperror.Error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, apperror.Wrap(err, "open_dir_error")
	}
	defer dir.Close()

	items, err := dir.Readdir(-1)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error")
	}

	files := make([]string, 0)
	for _, item := range items {
		if !item.IsDir() {
			files = append(files, item.Name())
		}
	}

	return files, nil
}

func BuildFileMD5Hash(path string) (string, apperror.Error) {
	file, err := os.Open(path)
	if err != nil {
		return "", apperror.Wrap(err, "file_open_error")
	}
	defer file.Close()

	// calculate the file size
	info, _ := file.Stat()
	filesize := info.Size()
	var filechunk float64 = 8192
	blocks := uint64(math.Ceil(float64(filesize) / filechunk))

	hash := md5.New()

	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(filechunk, float64(filesize-int64(i*uint64(filechunk)))))
		buf := make([]byte, blocksize)

		_, err := file.Read(buf)
		if err != nil {
			return "", apperror.Wrap(err, "file_read_error")
		}

		io.WriteString(hash, string(buf)) // append into the hash
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
