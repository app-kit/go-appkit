package files

import(
	"strings"
	"bufio"
	
	kit "github.com/theduke/go-appkit"
)
 
/**
 * Base file.
 */

// BaseFile that can be extended.
// You can use BaseFileIntID or BaseFileStrID in almost all cases.
type BaseFile struct {
	backend kit.ApiFileBackend

	backendName string
	bucket string
	
	name string
	extension string
	fullName string

	title string
	description string

	size int64
	mime string 	

	isImage bool

	width int
	height int
}

func (f *BaseFile) Collection() string {
	return "files"
}

func(f *BaseFile) Backend() kit.ApiFileBackend {
	return f.backend
}

func(f *BaseFile) SetBackend(x kit.ApiFileBackend) {
	f.backend = x
	f.backendName = x.Name()
}

func(f *BaseFile) BackendName() string {
	return f.backendName
}

func(f *BaseFile) SetBackendName(x string) {
	f.backendName = x
}

func(f *BaseFile) Bucket() string {
	return f.bucket
}

func(f *BaseFile) SetBucket(x string) {
	f.bucket = x
}



func(f *BaseFile) Name() string {
	return f.name
}

func(f *BaseFile) SetName(x string) {
	f.name = x
}

func(f *BaseFile) Extension() string {
	return f.extension
}

func(f *BaseFile) SetExtension(x string) {
	f.extension = x
}

func(f *BaseFile) FullName() string {
	return f.fullName
}

func(f *BaseFile) SetFullName(x string) {
	parts := strings.Split(x, ".")

	if len(parts) > 1 {
		f.name = strings.Join(parts[:len(parts) - 1], ".")
		f.extension = parts[len(parts) - 1]
	} else {
		f.name = x
		f.extension = ""
	}

	f.fullName = x
}


func(f *BaseFile) Title() string {
	return f.title
}

func(f *BaseFile) SetTitle(x string) {
	f.title = x
}

func(f *BaseFile) Description() string {
	return f.description
}

func(f *BaseFile) SetDescription(x string) {
	f.description = x
}


func(f *BaseFile) Size() int64 {
	return f.size
}

func(f *BaseFile) SetSize(x int64) {
	f.size = x
}

func(f *BaseFile) Mime() string {
	return f.mime
}

func(f *BaseFile) SetMime(x string) {
	f.mime = x
}

func(f *BaseFile) IsImage() bool {
	return f.isImage
}

func(f *BaseFile) SetIsImage(x bool) {
	f.isImage = x
}


func(f *BaseFile) Width() int {
	return f.width
}

func(f *BaseFile) SetWidth(x int) {
	f.width = x
}

func(f *BaseFile) Height() int {
	return f.height
}

func(f *BaseFile) SetHeight(x int) {
	f.height = x
}


/**
 * File with string id.
 */

type FileStrID struct {
	kit.BaseUserModelStrID
	BaseFile
}

// Ensure FileStrID implements ApiFile interface.
var _ kit.ApiFile = (*FileStrID)(nil)

func (f *FileStrID) Reader() (*bufio.Reader, kit.ApiError) {
	if f.backend == nil {
		return nil, nil
	}
	return f.backend.Reader(f)
}

func (f *FileStrID) Writer() (*bufio.Writer, kit.ApiError) {
	if f.backend == nil {
		return nil, nil
	}
	return f.backend.Writer(f)
}


/**
 * File with int id.
 */

type FileIntID struct {
	kit.BaseUserModelIntID
	BaseFile
}

// Ensure FileIntID implements ApiFile interface.
var _ kit.ApiFile = (*FileIntID)(nil)

func (f *FileIntID) Reader() (*bufio.Reader, kit.ApiError) {
	if f.backend == nil {
		return nil, nil
	}
	return f.backend.Reader(f)
}

func (f *FileIntID) Writer() (*bufio.Writer, kit.ApiError) {
	if f.backend == nil {
		return nil, nil
	}
	return f.backend.Writer(f)
}
