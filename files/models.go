package files

import (
	"io"
	"strings"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/users"
)

const (
	MEDIA_TYPE_IMAGE = "image"
	MEDIA_TYPE_VIDEO = "video"
)

/**
 * Base file.
 */

// File that can be extended.
// You can use FileIntID or FileStrID in almost all cases.
type File struct {
	Backend kit.FileBackend `db:"-"`

	BackendName string `db:"not-null;max:100"`
	BackendID   string `db:"not-null;max:150"`
	Bucket      string `db:"not-null;max:150"`

	// Used to store the tmp file path before it is persisted to the backend.
	TmpPath string

	Name      string `db:"not-null;max:1000"`
	Extension string `db:"max:100"`
	FullName  string `db:"not-null;max:1100"`

	Title       string `db:"max:100"`
	Description string `db:""`

	Size      int64
	Mime      string
	MediaType string

	IsImage bool

	Width  int
	Height int

	// Can be used for sorting.
	Weight int

	// Hash is an MD5 hash of the file contents.
	Hash string `db:"max:50"`

	// Can be used for categorization.
	Type string `db:"max:200"`

	// Stores any additional data required.
	Data map[string]interface{} `db:"marshal"`
}

func (f *File) Collection() string {
	return "files"
}

func (f *File) GetTmpPath() string {
	return f.TmpPath
}

func (f *File) SetTmpPath(x string) {
	f.TmpPath = x
}

func (f *File) GetBackend() kit.FileBackend {
	return f.Backend
}

func (f *File) SetBackend(x kit.FileBackend) {
	f.Backend = x
	f.BackendName = x.Name()
}

func (f *File) GetBackendName() string {
	return f.BackendName
}

func (f *File) SetBackendName(x string) {
	f.BackendName = x
}

func (f *File) GetBackendID() string {
	return f.BackendID
}

func (f *File) SetBackendID(x string) error {
	f.BackendID = x
	return nil
}

func (f *File) GetBucket() string {
	return f.Bucket
}

func (f *File) SetBucket(x string) {
	f.Bucket = x
}

func (f *File) GetName() string {
	return f.Name
}

func (f *File) SetName(x string) {
	f.Name = x
}

func (f *File) GetExtension() string {
	return f.Extension
}

func (f *File) SetExtension(x string) {
	f.Extension = x
}

func (f *File) GetFullName() string {
	return f.FullName
}

func (f *File) SetFullName(x string) {
	parts := strings.Split(x, ".")

	if len(parts) > 1 {
		f.Name = strings.Join(parts[:len(parts)-1], ".")
		f.Extension = parts[len(parts)-1]
	} else {
		f.Name = x
		f.Extension = ""
	}

	f.FullName = x
}

func (f *File) GetTitle() string {
	return f.Title
}

func (f *File) SetTitle(x string) {
	f.Title = x
}

func (f *File) GetDescription() string {
	return f.Description
}

func (f *File) SetDescription(x string) {
	f.Description = x
}

func (f *File) GetSize() int64 {
	return f.Size
}

func (f *File) SetSize(x int64) {
	f.Size = x
}

func (f *File) GetMime() string {
	return f.Mime
}

func (f *File) SetMime(x string) {
	f.Mime = x
}

func (f *File) GetMediaType() string {
	return f.MediaType
}

func (f *File) SetMediaType(x string) {
	f.MediaType = x
}

func (f *File) GetIsImage() bool {
	return f.IsImage
}

func (f *File) SetIsImage(x bool) {
	f.IsImage = x
}

func (f *File) GetWidth() int {
	return f.Width
}

func (f *File) SetWidth(x int) {
	f.Width = x
}

func (f *File) GetHeight() int {
	return f.Height
}

func (f *File) SetHeight(x int) {
	f.Height = x
}

func (f *File) GetHash() string {
	return f.Hash
}

func (f *File) SetHash(x string) {
	f.Hash = x
}

func (f *File) GetData() map[string]interface{} {
	return f.Data
}

func (f *File) SetData(x map[string]interface{}) {
	f.Data = x
}

func (f *File) GetType() string {
	return f.Type
}

func (f *File) SetType(x string) {
	f.Type = x
}

func (f *File) GetWeight() int {
	return f.Weight
}

func (f *File) SetWeight(x int) {
	f.Weight = x
}

type FileStrID struct {
	File
	db.StrIDModel
	users.StrUserModel

	ParentFile   *FileStrID
	ParentFileID string

	RelatedFiles []*FileStrID `db:"belongs-to:ID:ParentFileID"`
}

// Ensure FileStrID implements File interface.
var _ kit.File = (*FileStrID)(nil)

func (f *FileStrID) GetParentFile() kit.File {
	return f.ParentFile
}

func (f *FileStrID) SetParentFile(file kit.File) {
	f.ParentFile = file.(*FileStrID)
	f.ParentFileID = file.GetID().(string)
}

func (f *FileStrID) GetParentFileID() interface{} {
	return f.ParentFileID
}

func (f *FileStrID) SetParentFileID(id interface{}) {
	f.ParentFileID = id.(string)
}

func (f *FileStrID) GetRelatedFiles() []kit.File {
	files := make([]kit.File, 0)
	for _, file := range f.RelatedFiles {
		files = append(files, file)
	}

	return files
}

func (f *FileStrID) SetRelatedFiles(rawFiles []kit.File) {
	files := make([]*FileStrID, 0)
	for _, file := range rawFiles {
		files = append(files, file.(*FileStrID))
	}
	f.RelatedFiles = files
}

func (f *FileStrID) Reader() (kit.ReadSeekerCloser, apperror.Error) {
	if f.Backend == nil {
		return nil, nil
	}
	return f.Backend.Reader(f)
}

func (f *FileStrID) Writer(create bool) (string, io.WriteCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Called File.Writer() on a file with unset backend.")
	}
	return f.Backend.Writer(f, create)
}

/**
 * File with int id.
 */

type FileIntID struct {
	File
	db.IntIDModel
	users.IntUserModel

	ParentFile   *FileIntID
	ParentFileID uint64

	RelatedFiles []*FileIntID `db:"belongs-to:ID:ParentFileID"`
}

// Ensure FileIntID implements File interface.
var _ kit.File = (*FileIntID)(nil)

func (f *FileIntID) GetParentFile() kit.File {
	return f.ParentFile
}

func (f *FileIntID) SetParentFile(file kit.File) {
	f.ParentFile = file.(*FileIntID)
	f.ParentFileID = file.GetID().(uint64)
}

func (f *FileIntID) GetParentFileID() interface{} {
	return f.ParentFileID
}

func (f *FileIntID) SetParentFileID(id interface{}) {
	f.ParentFileID = id.(uint64)
}

func (f *FileIntID) GetRelatedFiles() []kit.File {
	files := make([]kit.File, 0)
	for _, file := range f.RelatedFiles {
		files = append(files, file)
	}

	return files
}

func (f *FileIntID) SetRelatedFiles(rawFiles []kit.File) {
	files := make([]*FileIntID, 0)
	for _, file := range rawFiles {
		files = append(files, file.(*FileIntID))
	}
	f.RelatedFiles = files
}

func (f *FileIntID) Reader() (kit.ReadSeekerCloser, apperror.Error) {
	if f.Backend == nil {
		return nil, nil
	}
	return f.Backend.Reader(f)
}

func (f *FileIntID) Writer(create bool) (string, io.WriteCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Called File.Writer() on a file with unset backend.")
	}
	return f.Backend.Writer(f, create)
}
