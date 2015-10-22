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
	"github.com/theduke/go-appkit/utils"
	db "github.com/theduke/go-dukedb"
)

type FileService struct {
	debug    bool
	registry kit.Registry

	resource       kit.Resource
	backends       map[string]kit.FileBackend
	defaultBackend kit.FileBackend
	model          kit.Model
}

// Ensure FileService implements FileService interface.
var _ kit.FileService = (*FileService)(nil)

func NewFileService(registry kit.Registry) *FileService {
	return &FileService{
		registry: registry,
		model:    &FileIntID{},
		backends: make(map[string]kit.FileBackend),
	}
}

func NewFileServiceWithFs(registry kit.Registry, dataPath string) *FileService {
	if dataPath == "" {
		panic("Empty data path")
	}

	service := NewFileService(registry)

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

func (s *FileService) Registry() kit.Registry {
	return s.registry
}

func (s *FileService) SetRegistry(x kit.Registry) {
	s.registry = x
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

func (h *FileService) Model() kit.Model {
	return h.model
}

func (h *FileService) SetModel(x kit.Model) {
	h.model = x
}

func (h FileService) BuildFileFromPath(bucket, path string, deleteFile bool) (kit.File, apperror.Error) {
	file := h.New()
	file.SetTmpPath(path)
	file.SetBucket(bucket)

	if err := h.BuildFile(file, nil, false, deleteFile); err != nil {
		return nil, err
	}

	return file, nil
}

func (h FileService) BuildFile(file kit.File, user kit.User, deleteDir, deleteFile bool) apperror.Error {
	if h.DefaultBackend == nil {
		return &apperror.Err{
			Code:    "no_default_backend",
			Message: "Cant build a file without a default backend.",
		}
	}

	filePath := file.GetTmpPath()
	if filePath == "" {
		return &apperror.Err{
			Code:    "no_tmp_path",
			Message: "You must set TmpPath on a file before building it.",
			Public:  true,
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

	file.SetBackend(backend)

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

	// Build the hash.
	hash, err2 := utils.BuildFileMD5Hash(filePath)
	if err2 != nil {
		return err2
	}

	file.SetHash(hash)

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
	if mimeType == "" {
		mimeType = mime.TypeByExtension("." + extension)
	}
	file.SetMime(mimeType)

	if strings.HasPrefix(mimeType, "image") {
		file.SetIsImage(true)
		file.SetMediaType(MEDIA_TYPE_IMAGE)

		// Determine image info.
		imageInfo, err := GetImageInfo(filePath)
		if imageInfo != nil {
			file.SetWidth(int(imageInfo.Width))
			file.SetHeight(int(imageInfo.Height))
		} else {
			h.Registry().Logger().Warningf("Could not determine image info: %v", err)
		}
	}

	if strings.HasPrefix(mimeType, "video") {
		file.SetMediaType(MEDIA_TYPE_VIDEO)
	}

	// Store the file in the backend.

	backendId, writer, err2 := file.Writer(true)
	if err2 != nil {
		return apperror.Wrap(err2, "file_backend_error")
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
	file.SetTmpPath("")

	if file.GetStrID() != "" {
		err2 = h.resource.Update(file, user)
	} else {
		err2 = h.resource.Create(file, user)
	}
	if err2 != nil {
		// Delete file from backend again.
		backend.DeleteFile(file)
		return apperror.Wrap(err2, "db_error", "Could not save file to database")
	}

	// Delete tmp file.
	if deleteFile {
		os.Remove(filePath)
	}

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

func (h *FileService) DeleteByID(id interface{}, user kit.User) apperror.Error {
	// Find the file first.
	f, err := h.Resource().FindOne(id)
	if err != nil {
		return err
	} else if f == nil {
		return apperror.New("not_found")
	}

	return h.Delete(f.(kit.File), user)
}

func (h *FileService) Delete(f kit.File, u kit.User) apperror.Error {
	// Delete file from backend.
	if f.GetBackendName() != "" && f.GetBackendID() != "" {
		backend := h.Backend(f.GetBackendName())
		if backend == nil {
			h.Registry().Logger().Errorf("Deleting file %v in backend %v, which is unconfigured", f.GetID(), f.GetBackendName())
		} else {
			if err := backend.DeleteFile(f); err != nil {
				return err
			}
		}
	}

	return h.resource.Delete(f, u)
}
