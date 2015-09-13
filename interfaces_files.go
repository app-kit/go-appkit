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
	GetBackend() ApiFileBackend
	SetBackend(ApiFileBackend)

	GetBackendName() string
	SetBackendName(string)

	GetBackendID() string
	SetBackendID(string) error

	// File bucket.
	GetBucket() string
	SetBucket(string)

	// File name without extension.
	GetName() string
	SetName(string)

	// File extension if available.
	GetExtension()  string
	SetExtension(string)

	// Name with extension.
	GetFullName() string
	SetFullName(string)

	GetTitle() string
	SetTitle(string)

	GetDescription() string
	SetDescription(string)

	// File size in bytes if available.	
	GetSize() int64
	SetSize(int64)

	// Mime type if available.
	GetMime() string
	SetMime(string)

	GetIsImage() bool
	SetIsImage(bool)

	// File width and hight in pixels for images and videos.
	GetWidth() int
	SetWidth(int)

	GetHeight() int
	SetHeight(int)

	// Get a reader for the file.
	// Might return an error if the file does not exist in the backend,
	// or it is not connected to a backend.
	Reader() (*bufio.Reader, ApiError)

	// Get a writer for the file.
	// Might return an error if the file is not connected to a backend.
	Writer(create bool) (string, *bufio.Writer, ApiError)
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
	Writer(f ApiFile, create bool) (string, *bufio.Writer, ApiError)
	// Retrieve a writer for a file in a bucket.
	WriterById(bucket, id string, create bool) (string, *bufio.Writer, ApiError)
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

	// Given a file instance with a specified bucket, read the file from filePath, upload it 
	// to the backend and then store it in the database.
	// If no file.GetBackendName() is empty, the default backend will be used.
	// The file will be deleted if everything succeeds. Otherwise, 
	// it will be left in the file system.
	// If deleteDir is true, the directory holding the file will be deleted 
	// also.
	BuildFile(file ApiFile, user ApiUser, filePath string, deleteDir bool) ApiError

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
