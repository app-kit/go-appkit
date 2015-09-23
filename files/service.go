package files

import (
	"fmt"
	"io"
	"mime"
	"os"
	"strings"

	"github.com/theduke/go-apperror"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/files/backends/fs"
	"github.com/theduke/go-appkit/resources"
	db "github.com/theduke/go-dukedb"
)

type FileService struct {
	debug bool
	deps  kit.Dependencies

	resource       kit.Resource
	backends       map[string]kit.FileBackend
	defaultBackend kit.FileBackend
	model          interface{}
}

// Ensure FileService implements FileService interface.
var _ kit.FileService = (*FileService)(nil)

func NewFileService(deps kit.Dependencies) *FileService {
	return &FileService{
		deps:     deps,
		model:    &FileIntID{},
		backends: make(map[string]kit.FileBackend),
	}
}

func NewFileServiceWithFs(deps kit.Dependencies, dataPath string) *FileService {
	if dataPath == "" {
		panic("Empty data path")
	}

	service := NewFileService(deps)

	res := resources.NewResource(&FileIntID{}, FilesResource{}, true)
	service.SetResource(res)

	fs, err := fs.New(dataPath)
	if err != nil {
		panic(fmt.Sprintf("Could not initialize filesystem backend: %v", err))
	}
	service.AddBackend(fs)

	return service
}

func (s *FileService) Debug() bool {
	return s.debug
}

func (s *FileService) SetDebug(x bool) {
	s.debug = x
}

func (s *FileService) Dependencies() kit.Dependencies {
	return s.deps
}

func (s *FileService) SetDependencies(x kit.Dependencies) {
	s.deps = x
}

func (h *FileService) Resource() kit.Resource {
	return h.resource
}

func (h *FileService) SetResource(x kit.Resource) {
	h.resource = x
}

func (h *FileService) Backend(name string) kit.FileBackend {
	return h.backends[name]
}

func (h *FileService) AddBackend(backend kit.FileBackend) {
	h.backends[backend.Name()] = backend

	if h.defaultBackend == nil {
		h.defaultBackend = backend
	}
}

func (h *FileService) DefaultBackend() kit.FileBackend {
	return h.defaultBackend
}

func (h *FileService) SetDefaultBackend(name string) {
	h.defaultBackend = h.backends[name]
}

func (h *FileService) Model() interface{} {
	return h.model
}

func (h *FileService) SetModel(x interface{}) {
	h.model = x
}

func (h FileService) BuildFile(file kit.File, user kit.User, filePath string, deleteDir bool) apperror.Error {
	if h.DefaultBackend == nil {
		return &apperror.Err{
			Code:    "no_default_backend",
			Message: "Cant build a file without a default backend.",
		}
	}

	if file.GetBackendName() == "" {
		file.SetBackendName(h.DefaultBackend().Name())
	}

	backend := h.Backend(file.GetBackendName())
	if backend == nil {
		return &apperror.Err{
			Code:    "unknown_backend",
			Message: fmt.Sprintf("The backend %v does not exist", file.GetBackendName()),
		}
	}

	if file.GetBucket() == "" {
		return &apperror.Err{
			Code:    "missing_bucket",
			Message: "Bucket must be set on the file",
		}
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		if err == os.ErrNotExist {
			return &apperror.Err{
				Code:    "file_not_found",
				Message: fmt.Sprintf("File %v does not exist", filePath),
			}
		}

		return apperror.Wrap(err, "stat_error",
			fmt.Sprintf("Could not get file stats for file at %v: %v", filePath, err))
	}

	if stat.IsDir() {
		return apperror.New("path_is_directory")
	}

	pathParts := strings.Split(filePath, string(os.PathSeparator))
	fullName := pathParts[len(pathParts)-1]
	nameParts := strings.Split(fullName, ".")

	// Determine extension.
	extension := ""
	if len(nameParts) > 1 {
		extension = nameParts[len(nameParts)-1]
	}

	file.SetFullName(fullName)
	file.SetSize(stat.Size())

	// Determine mime type.
	mimeType := GetMimeType(filePath)
	fmt.Printf("determined mime type: %v\n", mimeType)
	if mimeType == "" {
		mimeType = mime.TypeByExtension("." + extension)
	}
	file.SetMime(mimeType)

	// Determine image info.
	imageInfo, err := GetImageInfo(filePath)
	fmt.Printf("info: %+v, err: %v\n", imageInfo, err)
	if imageInfo != nil {
		file.SetIsImage(true)
		file.SetWidth(int(imageInfo.Width))
		file.SetHeight(int(imageInfo.Height))
	}

	// Store the file in the backend.
	backendId, writer, err2 := file.Writer(true)
	if err2 != nil {
		return apperror.Wrap(err2, "backend_error")
	}
	defer writer.Close()

	// Open file for reading.
	f, err := os.Open(filePath)
	if err != nil {
		return apperror.Wrap(err, "read_error", fmt.Sprintf("Could not read file at %v", filePath))
	}

	_, err = io.Copy(writer, f)
	if err != nil {
		f.Close()
		return apperror.Wrap(err, "copy_to_backend_failed")
	}
	f.Close()

	// File is stored in backend now!
	file.SetBackendID(backendId)

	// Persist file to db.
	err2 = h.resource.Create(file, user)
	if err2 != nil {
		// Delete file from backend again.
		backend.DeleteFile(file)
		return apperror.Wrap(err2, "db_error", "Could not save file to database")
	}

	// Delete tmp file.
	os.Remove(filePath)

	if deleteDir {
		dir := strings.Join(pathParts[:len(pathParts)-1], string(os.PathSeparator))
		os.RemoveAll(dir)
	}

	return nil
}

func (h *FileService) New() kit.File {
	f := h.resource.CreateModel().(kit.File)
	f.SetBackend(h.defaultBackend)
	return f
}

func (h *FileService) FindOne(id string) (kit.File, apperror.Error) {
	file, err := h.resource.FindOne(id)
	if err != nil {
		return nil, err
	} else if file == nil {
		return nil, nil
	} else {
		file := file.(kit.File)

		// Set backend on the file if found.
		if file.GetBackendName() != "" {
			if backend, ok := h.backends[file.GetBackendName()]; ok {
				file.SetBackend(backend)
			}
		}

		return file, nil
	}
}

func (h *FileService) Find(q db.Query) ([]kit.File, apperror.Error) {
	rawFiles, err := h.resource.Query(q)
	if err != nil {
		return nil, err
	}

	files := make([]kit.File, 0)
	for _, rawFile := range rawFiles {
		file := rawFile.(kit.File)

		// Set backend on the file if found.
		if file.GetBackendName() != "" {
			if backend, ok := h.backends[file.GetBackendName()]; ok {
				file.SetBackend(backend)
			}
		}

		files = append(files, file)
	}

	return files, nil
}

func (h *FileService) Create(f kit.File, u kit.User) apperror.Error {
	return h.resource.Create(f, u)
}

func (h *FileService) Update(f kit.File, u kit.User) apperror.Error {
	return h.resource.Update(f, u)
}

func (h *FileService) Delete(f kit.File, u kit.User) apperror.Error {
	return h.resource.Delete(f, u)
}
