package files

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"strings"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/users"
)

const (
	MEDIA_TYPE_IMAGE = "image"
	MEDIA_TYPE_VIdEO = "video"
)

/**
 * Base file.
 */

// File that can be extended.
// You can use FileIntId or FileStrId in almost all cases.
type File struct {
	Backend kit.FileBackend `db:"-"`

	BackendName string `db:"required;max:100"`
	BackendId   string `db:"required;max:150"`
	Bucket      string `db:"required;max:150"`

	// Used to store the tmp file path before it is persisted to the backend.
	TmpPath string

	Name      string `db:"required;max:1000"`
	Extension string `db:"max:100"`
	FullName  string `db:"required;max:1100"`

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

func (f *File) GetBackendId() string {
	return f.BackendId
}

func (f *File) SetBackendId(x string) error {
	f.BackendId = x
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

type FileStrId struct {
	File
	db.StrIdModel
	users.StrUserModel

	ParentFile   *FileStrId
	ParentFileId string

	RelatedFiles []*FileStrId `db:"belongs-to:Id:ParentFileId"`
}

// Ensure FileStrId implements File interface.
var _ kit.File = (*FileStrId)(nil)

func (f *FileStrId) GetParentFile() kit.File {
	return f.ParentFile
}

func (f *FileStrId) SetParentFile(file kit.File) {
	f.ParentFile = file.(*FileStrId)
	f.ParentFileId = file.GetId().(string)
}

func (f *FileStrId) GetParentFileId() interface{} {
	return f.ParentFileId
}

func (f *FileStrId) SetParentFileId(id interface{}) {
	f.ParentFileId = id.(string)
}

func (f *FileStrId) GetRelatedFiles() []kit.File {
	files := make([]kit.File, 0)
	for _, file := range f.RelatedFiles {
		files = append(files, file)
	}

	return files
}

func (f *FileStrId) SetRelatedFiles(rawFiles []kit.File) {
	files := make([]*FileStrId, 0)
	for _, file := range rawFiles {
		files = append(files, file.(*FileStrId))
	}
	f.RelatedFiles = files
}

// Note: needs to be duplicated for FileStrId because access to Id field is
// required.
func (f *FileStrId) Reader() (kit.ReadSeekerCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Can't call .Reader() on a file with empty backend.")
	}
	return f.Backend.Reader(f)
}

// Note: needs to be duplicated for FileStrId because access to Id field is
// required.
func (f *FileStrId) Base64() (string, apperror.Error) {
	if f.Backend == nil {
		panic("Can't call .Reader() on a file with unset backend.")
	}
	reader, err := f.Backend.Reader(f)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err2 := ioutil.ReadAll(reader)
	if err2 != nil {
		return "", apperror.Wrap(err2, "read_error")
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	return b64, nil
}

// Note: needs to be duplicated for FileStrId because access to Id field is
// required.
func (f *FileStrId) Writer(create bool) (string, io.WriteCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Called File.Writer() on a file with unset backend.")
	}
	return f.Backend.Writer(f, create)
}

/**
 * File with int id.
 */

type FileIntId struct {
	File
	db.IntIdModel
	users.IntUserModel

	ParentFile   *FileIntId
	ParentFileId uint64

	RelatedFiles []*FileIntId `db:"belongs-to:Id:ParentFileId"`
}

// Ensure FileIntId implements File interface.
var _ kit.File = (*FileIntId)(nil)

func (f *FileIntId) GetParentFile() kit.File {
	return f.ParentFile
}

func (f *FileIntId) SetParentFile(file kit.File) {
	f.ParentFile = file.(*FileIntId)
	f.ParentFileId = file.GetId().(uint64)
}

func (f *FileIntId) GetParentFileId() interface{} {
	return f.ParentFileId
}

func (f *FileIntId) SetParentFileId(id interface{}) {
	f.ParentFileId = id.(uint64)
}

func (f *FileIntId) GetRelatedFiles() []kit.File {
	files := make([]kit.File, 0)
	for _, file := range f.RelatedFiles {
		files = append(files, file)
	}

	return files
}

func (f *FileIntId) SetRelatedFiles(rawFiles []kit.File) {
	files := make([]*FileIntId, 0)
	for _, file := range rawFiles {
		files = append(files, file.(*FileIntId))
	}
	f.RelatedFiles = files
}

// Note: needs to be duplicated for FileIntId because access to Id field is
// required.
func (f *FileIntId) Reader() (kit.ReadSeekerCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Can't call .Reader() on a file with unset backend.")
	}
	return f.Backend.Reader(f)
}

// Note: needs to be duplicated for FileIntId because access to Id field is
// required.
func (f *FileIntId) Base64() (string, apperror.Error) {
	if f.Backend == nil {
		panic("Can't call .Reader() on a file with unset backend.")
	}
	reader, err := f.Backend.Reader(f)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err2 := ioutil.ReadAll(reader)
	if err2 != nil {
		return "", apperror.Wrap(err2, "read_error")
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	return b64, nil
}

// Note: needs to be duplicated for FileIntId because access to Id field is
// required.
func (f *FileIntId) Writer(create bool) (string, io.WriteCloser, apperror.Error) {
	if f.Backend == nil {
		panic("Called File.Writer() on a file with unset backend.")
	}
	return f.Backend.Writer(f, create)
}
