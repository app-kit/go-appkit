package files

import (
	"io"
	"strings"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/users"
)

/**
 * Base file.
 */

// BaseFile that can be extended.
// You can use BaseFileIntID or BaseFileStrID in almost all cases.
type BaseFile struct {
	Backend kit.ApiFileBackend `db:"-"`

	BackendName string
	BackendID   string
	Bucket      string

	Name      string
	Extension string
	FullName  string

	Title       string
	Description string

	Size int64
	Mime string

	IsImage bool

	Width  int
	Height int
}

func (f *BaseFile) Collection() string {
	return "files"
}

func (f *BaseFile) GetBackend() kit.ApiFileBackend {
	return f.Backend
}

func (f *BaseFile) SetBackend(x kit.ApiFileBackend) {
	f.Backend = x
	f.BackendName = x.Name()
}

func (f *BaseFile) GetBackendName() string {
	return f.BackendName
}

func (f *BaseFile) SetBackendName(x string) {
	f.BackendName = x
}

func (f *BaseFile) GetBackendID() string {
	return f.BackendID
}

func (f *BaseFile) SetBackendID(x string) error {
	f.BackendID = x
	return nil
}

func (f *BaseFile) GetBucket() string {
	return f.Bucket
}

func (f *BaseFile) SetBucket(x string) {
	f.Bucket = x
}

func (f *BaseFile) GetName() string {
	return f.Name
}

func (f *BaseFile) SetName(x string) {
	f.Name = x
}

func (f *BaseFile) GetExtension() string {
	return f.Extension
}

func (f *BaseFile) SetExtension(x string) {
	f.Extension = x
}

func (f *BaseFile) GetFullName() string {
	return f.FullName
}

func (f *BaseFile) SetFullName(x string) {
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

func (f *BaseFile) GetTitle() string {
	return f.Title
}

func (f *BaseFile) SetTitle(x string) {
	f.Title = x
}

func (f *BaseFile) GetDescription() string {
	return f.Description
}

func (f *BaseFile) SetDescription(x string) {
	f.Description = x
}

func (f *BaseFile) GetSize() int64 {
	return f.Size
}

func (f *BaseFile) SetSize(x int64) {
	f.Size = x
}

func (f *BaseFile) GetMime() string {
	return f.Mime
}

func (f *BaseFile) SetMime(x string) {
	f.Mime = x
}

func (f *BaseFile) GetIsImage() bool {
	return f.IsImage
}

func (f *BaseFile) SetIsImage(x bool) {
	f.IsImage = x
}

func (f *BaseFile) GetWidth() int {
	return f.Width
}

func (f *BaseFile) SetWidth(x int) {
	f.Width = x
}

func (f *BaseFile) GetHeight() int {
	return f.Height
}

func (f *BaseFile) SetHeight(x int) {
	f.Height = x
}

/**
 * File with string id.
 */

type FileStrID struct {
	users.BaseUserModelStrID
	BaseFile
}

// Ensure FileStrID implements ApiFile interface.
var _ kit.ApiFile = (*FileStrID)(nil)

func (f *FileStrID) Reader() (io.ReadCloser, kit.ApiError) {
	if f.Backend == nil {
		return nil, nil
	}
	return f.Backend.Reader(f)
}

func (f *FileStrID) Writer(create bool) (string, io.WriteCloser, kit.ApiError) {
	if f.Backend == nil {
		return "", nil, nil
	}
	return f.Backend.Writer(f, create)
}

/**
 * File with int id.
 */

type FileIntID struct {
	users.BaseUserModelIntID
	BaseFile
}

// Ensure FileIntID implements ApiFile interface.
var _ kit.ApiFile = (*FileIntID)(nil)

func (f *FileIntID) Reader() (io.ReadCloser, kit.ApiError) {
	if f.Backend == nil {
		return nil, nil
	}
	return f.Backend.Reader(f)
}

func (f *FileIntID) Writer(create bool) (string, io.WriteCloser, kit.ApiError) {
	if f.Backend == nil {
		return "", nil, nil
	}
	return f.Backend.Writer(f, create)
}
