package fs

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/theduke/go-apperror"

	kit "github.com/theduke/go-appkit"
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

func findUniqueFilePath(path string) (string, error) {
	if ok, err := FileExists(path); !ok {
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

		if ok, err := FileExists(path); !ok {
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

// Ensure Fs implements the FileBackend interface.
var _ kit.FileBackend = (*Fs)(nil)

func New(path string) (*Fs, apperror.Error) {
	fs := &Fs{
		name: "fs",
		path: path,
	}

	// Verify root path.
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, apperror.Wrap(err, "root_dir_initializiation_failed",
			fmt.Sprintf("Could not read or create the root path %v: ", path))
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

func (fs Fs) Buckets() ([]string, apperror.Error) {
	dir, err := os.Open(fs.path)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error")
	}
	defer dir.Close()

	dirItems, err := dir.Readdir(-1)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error")
	}

	buckets := make([]string, 0)

	for _, item := range dirItems {
		if item.IsDir() && item.Name() != "." && item.Name() != ".." {
			buckets = append(buckets, item.Name())
		}
	}

	return buckets, nil
}

func (fs Fs) HasBucket(bucket string) (bool, apperror.Error) {
	f, err := os.Open(fs.bucketPath(bucket))
	if err != nil {
		// Todo: check for "does not exist" error and return other
		// errors.
		return false, nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, apperror.Wrap(err, "read_error")
	}

	if info.IsDir() {
		return true, nil
	} else {
		return false, nil
	}
}

func (fs Fs) CreateBucket(bucket string, _ kit.BucketConfig) apperror.Error {
	if err := os.MkdirAll(fs.bucketPath(bucket), 0777); err != nil {
		return apperror.Wrap(err, "create_bucket_failed")
	}

	return nil
}

func (fs Fs) DeleteBucket(bucket string) apperror.Error {
	if err := os.RemoveAll(fs.bucketPath(bucket)); err != nil {
		return apperror.Wrap(err, "bucket_delete_failed",
			fmt.Sprintf("Could not delete bucket %v", bucket))
	}

	return nil
}

func (fs Fs) BucketConfig(string) kit.BucketConfig {
	// FS does not support any bucket configuration.
	return nil
}

func (fs Fs) ConfigureBucket(string, kit.BucketConfig) apperror.Error {
	// FS does not support any bucket configuration.
	return nil
}

func (fs Fs) ClearBucket(bucket string) apperror.Error {
	files, err := fs.FileIDs(bucket)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := os.Remove(fs.filePath(bucket, file)); err != nil {
			return apperror.Wrap(err, "delete_failed",
				fmt.Sprintf("Could not delete file %v from bucket %v", file, bucket))
		}
	}

	return nil
}

func (fs Fs) ClearAll() apperror.Error {
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

func (fs Fs) FileIDs(bucket string) ([]string, apperror.Error) {
	bucketPath := fs.bucketPath(bucket)
	dir, err := os.Open(bucketPath)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error")
	}
	defer dir.Close()

	items, err := dir.Readdir(-1)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error")
	}

	ids := make([]string, 0)
	for _, item := range items {
		if !item.IsDir() {
			ids = append(ids, item.Name())
		}
	}

	return ids, nil
}

func (fs Fs) HasFile(f kit.File) (bool, apperror.Error) {
	return fs.HasFileById(f.GetBucket(), f.GetBackendID())
}

func (fs Fs) HasFileById(bucket, id string) (bool, apperror.Error) {
	path := fs.filePath(bucket, id)
	if f, err := os.Open(path); err != nil {
		// Todo: check for other errors.
		return false, nil
	} else {
		f.Close()
		return true, nil
	}
}

func (fs Fs) DeleteFile(f kit.File) apperror.Error {
	return fs.DeleteFileById(f.GetBucket(), f.GetBackendID())
}

func (fs Fs) DeleteFileById(bucket, id string) apperror.Error {
	path := fs.filePath(bucket, id)
	if err := os.Remove(path); err != nil {
		return apperror.Wrap(err, "file_delete_failed",
			fmt.Sprintf("Could not delete file %v from bucket %v", bucket, id))
	}

	return nil
}

func (fs Fs) Reader(f kit.File) (io.ReadCloser, apperror.Error) {
	return fs.ReaderById(f.GetBucket(), f.GetBackendID())
}

func (fs Fs) ReaderById(bucket, id string) (io.ReadCloser, apperror.Error) {
	if id == "" {
		return nil, apperror.New("empty_file_id")
	}

	path := fs.filePath(bucket, id)
	f, err := os.Open(path)
	if err != nil {
		return nil, apperror.Wrap(err, "read_error",
			fmt.Sprintf("Could not open file %v", path))
	}

	return f, nil
}

func (fs Fs) Writer(f kit.File, create bool) (string, io.WriteCloser, apperror.Error) {
	id := f.GetBackendID()
	if create {
		id = f.GetFullName()
	}
	return fs.WriterById(f.GetBucket(), id, create)
}

func (fs Fs) WriterById(bucket, id string, create bool) (string, io.WriteCloser, apperror.Error) {
	if id == "" {
		return "", nil, apperror.New("empty_file_id")
	}

	if flag, err := fs.HasBucket(bucket); err != nil {
		return "", nil, err
	} else if !flag {
		if create {
			if err := fs.CreateBucket(bucket, nil); err != nil {
				return "", nil, err
			}
		} else {
			return "", nil, &apperror.Err{
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
			return "", nil, apperror.Wrap(err, "read_error")
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return "", nil, apperror.Wrap(err, "create_failed",
			fmt.Sprintf("Could not create file %v", path))
	}

	pathParts := strings.Split(path, string(os.PathSeparator))
	name := pathParts[len(pathParts)-1]

	return name, f, nil
}
