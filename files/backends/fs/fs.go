package fs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	kit "github.com/theduke/go-appkit"
)

func fileExists(path string) (bool, error) {
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

func findUniqueFilePath(path string) (string, error) {
	if ok, err := fileExists(path); !ok {
		return path, nil
	} else if err != nil {
		return "", err
	}

	// File already exists.

	pathParts := strings.Split(path, string(os.PathSeparator))
	dir := strings.Join(pathParts[:len(pathParts)-1], string(os.PathSeparator))

	name := pathParts[len(pathParts)-1]
	extension := ""
	index := 1

	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		name = strings.Join(parts[:len(parts)-1], ".")
		extension = parts[len(parts)-1]
	}

	for {
		path = dir + string(os.PathSeparator) + name + "_" + strconv.Itoa(index) + "." + extension

		if ok, err := fileExists(path); !ok {
			// Found non-existant file name!
			break
		} else if err != nil {
			return "", err
		} else {
			index++
		}
	}

	return path, nil
}

type Fs struct {
	name string
	path string
}

// Ensure Fs implements the ApiFileBackend interface.
var _ kit.ApiFileBackend = (*Fs)(nil)

func New(path string) (*Fs, kit.ApiError) {
	fs := &Fs{
		name: "fs",
		path: path,
	}

	// Verify root path.
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, kit.Error{
			Code:    "root_dir_initializiation_failed",
			Message: fmt.Sprintf("Could not read or create the root path %v: ", path, err.Error()),
		}
	}

	return fs, nil
}

func (fs *Fs) Name() string {
	return fs.name
}

func (fs *Fs) SetName(x string) {
	fs.name = x
}

func (fs Fs) bucketPath(bucket string) string {
	return fs.path + string(os.PathSeparator) + bucket
}

func (fs Fs) filePath(bucket, file string) string {
	return fs.bucketPath(bucket) + string(os.PathSeparator) + file
}

func (fs Fs) Buckets() ([]string, kit.ApiError) {
	dir, err := os.Open(fs.path)
	if err != nil {
		return nil, kit.Error{
			Code:    "read_error",
			Message: err.Error(),
		}
	}
	defer dir.Close()

	dirItems, err := dir.Readdir(-1)
	if err != nil {
		return nil, kit.Error{
			Code:    "read_error",
			Message: err.Error(),
		}
	}

	buckets := make([]string, 0)

	for _, item := range dirItems {
		if item.IsDir() && item.Name() != "." && item.Name() != ".." {
			buckets = append(buckets, item.Name())
		}
	}

	return buckets, nil
}

func (fs Fs) HasBucket(bucket string) (bool, kit.ApiError) {
	f, err := os.Open(fs.bucketPath(bucket))
	if err != nil {
		// Todo: check for "does not exist" error and return other
		// errors.
		return false, nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, kit.Error{
			Code:    "read_error",
			Message: err.Error(),
		}
	}

	if info.IsDir() {
		return true, nil
	} else {
		return false, nil
	}
}

func (fs Fs) CreateBucket(bucket string, _ kit.ApiBucketConfig) kit.ApiError {
	if err := os.Mkdir(fs.bucketPath(bucket), 0777); err != nil {
		return kit.Error{
			Code:    "create_bucket_failed",
			Message: err.Error(),
		}
	}

	return nil
}

func (fs Fs) DeleteBucket(bucket string) kit.ApiError {
	if err := os.RemoveAll(fs.bucketPath(bucket)); err != nil {
		return kit.Error{
			Code:    "bucket_delete_failed",
			Message: fmt.Sprintf("Could not delete bucket %v: %v", bucket, err),
		}
	}

	return nil
}

func (fs Fs) BucketConfig(string) kit.ApiBucketConfig {
	// FS does not support any bucket configuration.
	return nil
}

func (fs Fs) ConfigureBucket(string, kit.ApiBucketConfig) kit.ApiError {
	// FS does not support any bucket configuration.
	return nil
}

func (fs Fs) ClearBucket(bucket string) kit.ApiError {
	files, err := fs.FileIDs(bucket)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := os.Remove(fs.filePath(bucket, file)); err != nil {
			return kit.Error{
				Code: "delete_failed",
				Message: fmt.Sprintf(
					"Could not delete file %v from bucket %v: %v", file, bucket, err),
			}
		}
	}

	return nil
}

func (fs Fs) ClearAll() kit.ApiError {
	buckets, err := fs.Buckets()
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		if err := fs.ClearBucket(bucket); err != nil {
			return err
		}
	}

	return nil
}

func (fs Fs) FileIDs(bucket string) ([]string, kit.ApiError) {
	bucketPath := fs.bucketPath(bucket)
	dir, err := os.Open(bucketPath)
	if err != nil {
		return nil, kit.Error{
			Code:    "read_error",
			Message: err.Error(),
		}
	}
	defer dir.Close()

	items, err := dir.Readdir(-1)
	if err != nil {
		return nil, kit.Error{
			Code:    "read_error",
			Message: err.Error(),
		}
	}

	ids := make([]string, 0)
	for _, item := range items {
		if !item.IsDir() {
			ids = append(ids, item.Name())
		}
	}

	return ids, nil
}

func (fs Fs) HasFile(f kit.ApiFile) (bool, kit.ApiError) {
	return fs.HasFileById(f.GetBucket(), f.GetFullName())
}

func (fs Fs) HasFileById(bucket, id string) (bool, kit.ApiError) {
	path := fs.filePath(bucket, id)
	if f, err := os.Open(path); err != nil {
		// Todo: check for other errors.
		return false, nil
	} else {
		f.Close()
		return true, nil
	}
}

func (fs Fs) DeleteFile(f kit.ApiFile) kit.ApiError {
	return fs.DeleteFileById(f.GetBucket(), f.GetFullName())
}

func (fs Fs) DeleteFileById(bucket, id string) kit.ApiError {
	path := fs.filePath(bucket, id)
	if err := os.Remove(path); err != nil {
		return kit.Error{
			Code:    "file_delete_failed",
			Message: fmt.Sprintf("Could not delete file %v from bucket %v: %v", bucket, id, err),
		}
	}

	return nil
}

func (fs Fs) Reader(f kit.ApiFile) (*bufio.Reader, kit.ApiError) {
	return fs.ReaderById(f.GetBucket(), f.GetBackendID())
}

func (fs Fs) ReaderById(bucket, id string) (*bufio.Reader, kit.ApiError) {
	if id == "" {
		return nil, kit.Error{Code: "empty_file_id"}
	}

	path := fs.filePath(bucket, id)
	f, err := os.Open(path)
	if err != nil {
		return nil, kit.Error{
			Code:    "read_error",
			Message: fmt.Sprintf("Could not open file %v: %v", path, err),
		}
	}

	return bufio.NewReader(f), nil
}

func (fs Fs) Writer(f kit.ApiFile, create bool) (string, *bufio.Writer, kit.ApiError) {
	return fs.WriterById(f.GetBucket(), f.GetFullName(), create)
}

func (fs Fs) WriterById(bucket, id string, create bool) (string, *bufio.Writer, kit.ApiError) {
	if id == "" {
		return "", nil, kit.Error{Code: "empty_file_id"}
	}

	if flag, err := fs.HasBucket(bucket); err != nil {
		return "", nil, err
	} else if !flag {
		if create {
			if err := fs.CreateBucket(bucket, nil); err != nil {
				return "", nil, err
			}
		} else {
			return "", nil, kit.Error{
				Code:    "unknown_bucket",
				Message: fmt.Sprintf("Trying to get writer for file %v in non-existant bucket %v", id, bucket),
			}
		}
	}

	path := fs.filePath(bucket, id)

	// When creating, check if a file with the same name already exists,
	// and if so, append _x to the name.
	if create {
		var err error
		path, err = findUniqueFilePath(path)
		if err != nil {
			return "", nil, kit.Error{
				Code:    "read_error",
				Message: err.Error(),
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return "", nil, kit.Error{
			Code:    "create_failed",
			Message: fmt.Sprintf("Could not create file %v: %v", path, err),
		}
	}

	pathParts := strings.Split(path, string(os.PathSeparator))
	name := pathParts[len(pathParts)-1]

	return name, bufio.NewWriter(f), nil
}
