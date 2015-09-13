package files

import (
	"fmt"
	"os"
	"mime"
	"strings"
	"io"
	
	db "github.com/theduke/go-dukedb"	
	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/files/backends/fs"
)

type FileHandler struct {
	app *kit.App
	model interface{}
	resource kit.ApiResource
	backends map[string]kit.ApiFileBackend
	defaultBackend kit.ApiFileBackend
}

// Ensure FileHandler implements ApiFileHandler interface.
var _ kit.ApiFileHandler = (*FileHandler)(nil)

func NewFileHandler() *FileHandler {
	return &FileHandler{
		model: &FileIntID{},
		backends: make(map[string]kit.ApiFileBackend),
	}
}

func NewFileHandlerWithFs(dataPath string) *FileHandler {
	if dataPath == "" {
		panic("Empty data path")
	}

	handler := NewFileHandler()

	res := kit.NewResource(&FileIntID{}, FilesResource{})
	handler.SetResource(res)

	fs, err := fs.New(dataPath)
	if err != nil {
		panic(fmt.Sprintf("Could not initialize filesystem backend: %v", err))
	}
	handler.AddBackend(fs)

	return handler
}

func (h *FileHandler) SetApp(app *kit.App) {
	h.app = app
}

func(h *FileHandler) Resource() kit.ApiResource {
	return h.resource
}

func(h *FileHandler) SetResource(x kit.ApiResource) {
	h.resource = x
}

func (h *FileHandler) Backend(name string) kit.ApiFileBackend {
	return h.backends[name]
}

func (h *FileHandler) AddBackend(backend kit.ApiFileBackend) {
	h.backends[backend.Name()] = backend

	if h.defaultBackend == nil {
		h.defaultBackend = backend
	}
}

func (h *FileHandler) DefaultBackend() kit.ApiFileBackend {
	return h.defaultBackend
}

func (h *FileHandler) SetDefaultBackend(name string) {
	h.defaultBackend = h.backends[name]
}

func(h *FileHandler) Model() interface{} {
	return h.model
}

func(h *FileHandler) SetModel(x interface{}) {
	h.model = x
}

func (h FileHandler) BuildFile(file kit.ApiFile, user kit.ApiUser, filePath string, deleteDir bool) kit.ApiError {
	if h.DefaultBackend == nil {
		return kit.Error{
			Code: "no_default_backend",
			Message: "Cant build a file without a default backend.",
		}
	}

	if file.GetBackendName() == "" {
		file.SetBackendName(h.DefaultBackend().Name())
	}

	backend := h.Backend(file.GetBackendName())
	if backend == nil {
		return kit.Error{
			Code: "unknown_backend",
			Message: fmt.Sprintf("The backend %v does not exist", file.GetBackendName()),
		}
	}

	if file.GetBucket() == "" {
		return kit.Error{
			Code: "missing_bucket",
			Message: "Bucket must be set on the file",
		}
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		if err == os.ErrNotExist {
			return kit.Error{
				Code: "file_not_found",
				Message: fmt.Sprintf("File %v does not exist", filePath),
			}
		}

		return kit.Error{
			Code: "stat_error",
			Message: fmt.Sprintf("Could not get file stats for file at %v: %v", filePath, err),
			Errors: []error{err},
		}
	}

	if stat.IsDir() {
		return kit.Error{Code: "path_is_directory"}
	}

	pathParts := strings.Split(filePath, string(os.PathSeparator))
	fullName := pathParts[len(pathParts) - 1]
	nameParts := strings.Split(fullName, ".")
	extension := ""

	if len(nameParts) > 1 {
		extension = nameParts[len(nameParts) - 1]
	}

	file.SetFullName(fullName)
	file.SetSize(stat.Size())
	file.SetMime(mime.TypeByExtension("." + extension))

	// Todo: isImage, width, height

	// Store the file in the backend.
	backendId, writer, err2 := file.Writer(true)
	if err2 != nil {
		return kit.Error{
			Code: "backend_error",
			Message: err2.Error(),
		}
	}

	// Open file for reading.
	f, err := os.Open(filePath)
	if err != nil {
		return kit.Error{
			Code: "read_error",
			Message: fmt.Sprintf("Could not read file at %v", filePath),
		}
	}

	_, err = io.Copy(writer, f)
	if err != nil {
		f.Close()
		return kit.Error{
			Code: "copy_to_backend_failed",
			Message: err.Error(),
		}
	}
	f.Close()

	// File is stored in backend now!
	file.SetBackendID(backendId)

	// Persist file to db.
	err2 = h.resource.Create(file, user)
	if err2 != nil {
		// Delete file from backend again.
		backend.DeleteFile(file)
		return kit.Error{
			Code: "db_error",
			Message: fmt.Sprintf("Could not save file to database: %v\n", err2),
			Errors: []error{err2},
		}
	}

	// Delete tmp file.
	os.Remove(filePath)

	if deleteDir {
		dir := strings.Join(pathParts[:len(pathParts) - 1], string(os.PathSeparator))
		os.RemoveAll(dir)
	}

	return nil
}

func (h *FileHandler) New() kit.ApiFile {
	f := h.resource.NewModel().(kit.ApiFile)
	f.SetBackend(h.defaultBackend)
	return f
}

func (h *FileHandler) FindOne(id string) (kit.ApiFile, kit.ApiError) {
	file, err := h.resource.FindOne(id)
	if err != nil {
		return nil, err
	} else if file == nil {
		return nil, nil
	} else {
		file := file.(kit.ApiFile)

		// Set backend on the file if found.
		if file.GetBackendName() != "" {
			if backend, ok := h.backends[file.GetBackendName()]; ok {
				file.SetBackend(backend)
			}
		}

		return file, nil
	}
}

func (h *FileHandler) Find(q *db.Query) ([]kit.ApiFile, kit.ApiError) {
	rawFiles, err := h.resource.Find(q)
	if err != nil {
		return nil, err
	}

	files := make([]kit.ApiFile, 0)
	for _, rawFile := range rawFiles {
		file := rawFile.(kit.ApiFile)

		// Set backend on the file if found.
		if file.GetBackendName() != "" {
			if backend, ok := h.backends[file.GetBackendName()]; ok {
				file.SetBackend(backend)
			}
		}

		files = append(files, file)
	}

	return files,  nil
}

func (h *FileHandler) Create(f kit.ApiFile, u kit.ApiUser) kit.ApiError {
	return h.resource.Create(f, u)
}

func (h *FileHandler) Update(f kit.ApiFile, u kit.ApiUser) kit.ApiError {
	return h.resource.Update(f, u)
}

func (h *FileHandler) Delete(f kit.ApiFile, u kit.ApiUser) kit.ApiError {
	return h.resource.Delete(f, u)
}
