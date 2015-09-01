package files

import (
	"fmt"
	
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
		if file.BackendName() != "" {
			if backend, ok := h.backends[file.BackendName()]; ok {
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
		if file.BackendName() != "" {
			if backend, ok := h.backends[file.BackendName()]; ok {
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
