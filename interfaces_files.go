package appkit

import (
	"io"

	db "github.com/theduke/go-dukedb"

	. "github.com/theduke/go-appkit/error"
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
	GetExtension() string
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
	Reader() (io.ReadCloser, Error)

	// Get a writer for the file.
	// Might return an error if the file is not connected to a backend.
	Writer(create bool) (string, io.WriteCloser, Error)
}

type ApiFileBackend interface {
	Name() string
	SetName(string)

	// Lists the buckets that currently exist.
	Buckets() ([]string, Error)

	// Check if a Bucket exists.
	HasBucket(string) (bool, Error)

	// Create a bucket.
	CreateBucket(string, ApiBucketConfig) Error

	// Return the configuration for a a bucket.
	BucketConfig(string) ApiBucketConfig

	// Change the configuration for a bucket.
	ConfigureBucket(string, ApiBucketConfig) Error

	// Delete all files in a bucket.
	ClearBucket(bucket string) Error

	DeleteBucket(bucket string) Error

	// Clear all buckets.
	ClearAll() Error

	// Return the ids of all files in a bucket.
	FileIDs(bucket string) ([]string, Error)

	HasFile(ApiFile) (bool, Error)
	HasFileById(bucket, id string) (bool, Error)

	DeleteFile(ApiFile) Error
	DeleteFileById(bucket, id string) Error

	// Retrieve a reader for a file.
	Reader(ApiFile) (io.ReadCloser, Error)
	// Retrieve a reader for a file in a bucket.
	ReaderById(bucket, id string) (io.ReadCloser, Error)

	// Retrieve a writer for a file in a bucket.
	Writer(f ApiFile, create bool) (string, io.WriteCloser, Error)
	// Retrieve a writer for a file in a bucket.
	WriterById(bucket, id string, create bool) (string, io.WriteCloser, Error)
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
	BuildFile(file ApiFile, user ApiUser, filePath string, deleteDir bool) Error

	// Resource callthroughs.
	// The following methods map resource methods for convenience.

	// Create a new file model.
	New() ApiFile

	FindOne(id string) (ApiFile, Error)
	Find(*db.Query) ([]ApiFile, Error)

	Create(ApiFile, ApiUser) Error
	Update(ApiFile, ApiUser) Error
	Delete(ApiFile, ApiUser) Error
}
