package files

import (
	db "github.com/theduke/go-dukedb"	
	kit "github.com/theduke/go-appkit"	
)

type FileHandler struct {
	resource kit.ApiResource
	backends map[string]kit.ApiFileBackend
	defaultBackend kit.ApiFileBackend
	model interface{}
}

// Ensure FileHandler implements ApiFileHandler interface.
var _ kit.ApiFileHandler = (*FileHandler)(nil)

func(h *FileHandler) Resource() kit.ApiResource {
	return h.resource
}

func(h *FileHandler) SetResource(x kit.ApiResource) {
	h.resource = x
}

func (h *FileHandler) Backend(name string) kit.ApiFileBackend {
	return h.backends[name]
}

func (h *FileHandler) AddBackend(name string, backend kit.ApiFileBackend) {
	backend.SetName(name)
	h.backends[name] = backend

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
