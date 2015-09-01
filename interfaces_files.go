package appkit

import (
	"bufio"

	db "github.com/theduke/go-dukedb"
)

type ApiBucketConfig interface {

}

// Interface for a File stored in a database backend.
type ApiFile interface {
	// File can belong to a user.
	ApiUserModel

	// Retrieve the backend the file is stored in or should be stored in.
	// WARNING: can return nil.
	Backend() ApiFileBackend
	SetBackend(ApiFileBackend)

	BackendName() string
	SetBackendName(string)

	// File bucket.
	Bucket() string
	SetBucket(string)

	// File name without extension.
	Name() string
	SetName(string)

	// File extension if available.
	Extension()  string
	SetExtension(string)

	// Name with extension.
	FullName() string
	SetFullName(string)

	Title() string
	SetTitle(string)

	Description() string
	SetDescription(string)

	// File size in bytes if available.	
	Size() int64
	SetSize(int64)

	// Mime type if available.
	Mime() string
	SetMime(string)

	IsImage() bool
	SetIsImage(bool)

	// File width and hight in pixels for images and videos.
	Width() int
	SetWidth(int)

	Height() int
	SetHeight(int)

	// Get a reader for the file.
	// Might return an error if the file does not exist in the backend,
	// or it is not connected to a backend.
	Reader() (*bufio.Reader, ApiError)

	// Get a writer for the file.
	// Might return an error if the file is not connected to a backend.
	Writer() (*bufio.Writer, ApiError)
}

type ApiFileBackend interface {
	Name() string
	SetName(string)

	// Lists the buckets that currently exist.	
	Buckets() ([]string, ApiError)

	// Check if a Bucket exists.
	HasBucket(string) (bool, ApiError)

	// Create a bucket.
	CreateBucket(string, ApiBucketConfig) ApiError

	// Return the configuration for a a bucket.
	BucketConfig(string) ApiBucketConfig

	// Change the configuration for a bucket.
	ConfigureBucket(string, ApiBucketConfig) ApiError

	// Delete all files in a bucket.
	ClearBucket(bucket string) ApiError

	DeleteBucket(bucket string) ApiError

	// Clear all buckets.
	ClearAll() ApiError

	// Return the ids of all files in a bucket.
	FileIDs(bucket string) ([]string, ApiError)

	HasFile(ApiFile) (bool, ApiError)
	HasFileById(bucket, id string) (bool, ApiError)

	DeleteFile(ApiFile) ApiError
	DeleteFileById(bucket, id string) ApiError

	// Retrieve a reader for a file.
	Reader(ApiFile) (*bufio.Reader, ApiError)
	// Retrieve a reader for a file in a bucket.
	ReaderById(bucket, id string) (*bufio.Reader, ApiError)

	// Retrieve a writer for a file in a bucket.
	Writer(ApiFile) (*bufio.Writer, ApiError)
	// Retrieve a writer for a file in a bucket.
	WriterById(bucket, id string) (*bufio.Writer, ApiError)
}

type ApiFileHandler interface {
	SetApp(*App)

	Resource() ApiResource
	SetResource(ApiResource)

	Backend(string) ApiFileBackend
	AddBackend(ApiFileBackend)

	DefaultBackend() ApiFileBackend
	SetDefaultBackend(string)

	Model() interface{}
	SetModel(interface{})

	// Resource callthroughs.
	// The following methods map resource methods for convenience.

	// Create a new file model.
	New() ApiFile

	FindOne(id string) (ApiFile, ApiError)
	Find(*db.Query) ([]ApiFile, ApiError)

	Create(ApiFile, ApiUser) ApiError
	Update(ApiFile, ApiUser) ApiError
	Delete(ApiFile, ApiUser) ApiError
}
