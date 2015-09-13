package files

import (
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
)

type FilesResource struct{}

func getTmpPath(res kit.ApiResource) string {
	tmpPath := res.App().Config.UString("tmpDirUploads")
	if tmpPath == "" {
		tmpPath = res.App().Config.UString("tmpDir")
		if tmpPath != "" {
			tmpPath += string(os.PathSeparator) + "uploads"
		}
	}

	return tmpPath
}

func (_ FilesResource) ApiCreate(res kit.ApiResource, obj db.Model, r kit.ApiRequest) kit.ApiResponse {
	tmpPath := getTmpPath(res)
	if tmpPath == "" {
		return &kit.Response{
			Error: kit.Error{
				Code:    "no_tmp_path",
				Message: "Tmp path is not configured",
			},
		}
	}

	tmpFile := r.GetMeta().String("file")
	if tmpFile == "" {
		return &kit.Response{
			Error: kit.Error{
				Code:    "missing_file_in_meta",
				Message: "Expected 'file' in metadata with id of tmp file",
			},
		}
	}

	tmpPath = tmpPath + string(os.PathSeparator) + tmpFile

	user := r.GetUser()
	if allowCreate, ok := r.(kit.AllowCreateHook); ok {
		if !allowCreate.AllowCreate(res, obj, user) {
			return kit.NewErrorResponse("permission_denied", "")
		}
	}

	file := obj.(kit.ApiFile)
	err := res.App().FileHandler().BuildFile(file, user, tmpPath, true)

	err = res.Create(obj, user)
	if err != nil {
		return &kit.Response{Error: err}
	}

	return &kit.Response{
		Data: obj,
	}
}

func handleUpload(a *kit.App, tmpPath string, r *http.Request) ([]string, kit.ApiError) {
	a.Logger.Info("handling upload")
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, kit.Error{Code: "multipart_error", Message: err.Error()}
	}

	files := make([]string, 0)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, kit.Error{
					Code:    "read_error",
					Message: err.Error(),
				}
			}
		}

		filename := part.FileName()
		if filename == "" {
			// Not a file?
			continue
		}

		id := uuid.NewV4().String()
		path := tmpPath + string(os.PathSeparator) + id

		if err := os.MkdirAll(path, 0777); err != nil {
			return nil, kit.Error{
				Code:    "create_dir_failed",
				Message: err.Error(),
			}
		}

		filename = kit.Canonicalize(filename)
		if filename == "" {
			filename = id
		}

		filePath := path + string(os.PathSeparator) + filename

		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return nil, kit.Error{
				Code:    "file_create_failed",
				Message: err.Error(),
			}
		}

		_, err = io.Copy(file, part)
		if err != nil {
			return nil, kit.Error{
				Code:    "file_create_failed",
				Message: err.Error(),
			}
		}

		files = append(files, id+string(os.PathSeparator)+filename)
	}

	return files, nil
}

func (hooks FilesResource) HttpRoutes(res kit.ApiResource) []*kit.HttpRoute {
	routes := make([]*kit.HttpRoute, 0)

	// Upload route.
	uploadOptionsRoute := &kit.HttpRoute{
		Route: "/api/files/upload",
		Method: "OPTIONS", 
		Handler: func(a *kit.App, r kit.ApiRequest, w http.ResponseWriter) (kit.ApiResponse, bool) {
			header := w.Header()

			allowedOrigins := a.Config.UString("fileHandler.allowedOrigins", "*")
			header.Set("Access-Control-Allow-Origin", allowedOrigins)

			header.Set("Access-Control-Allow-Methods", "OPTIONS, POST")

			allowedHeaders := a.Config.UString("accessControl.allowedHeaders")
			if allowedHeaders == "" {
				allowedHeaders = "Authentication, Content-Type, Content-Range, Content-Disposition"
			} else {
				allowedHeaders += ", Authentication, Content-Type, Content-Range, Content-Disposition"
			}
			header.Set("Access-Control-Allow-Headers", allowedHeaders)

			w.WriteHeader(200)
			return nil, true
		},
	}
	routes = append(routes, uploadOptionsRoute)

	tmpPath := getTmpPath(res)
	if tmpPath == "" {
		panic("Empty tmp path")
	}

	uploadRoute := &kit.HttpRoute{
		Route: "/api/files/upload",
		Method: "POST",
		Handler: func(a *kit.App, r kit.ApiRequest, w http.ResponseWriter) (kit.ApiResponse, bool) {

			if a.Config.UBool("fileHandler.requiresAuth", false) {
				if r.GetUser() == nil {
					return kit.NewErrorResponse("permission_denied", ""), false
				}
			}

			var files []string
			var err kit.ApiError

			if err == nil {
				files, err = handleUpload(a, tmpPath, r.GetHttpRequest())
				if err != nil {
					return &kit.Response{Error: err}, false
				}
			}

			data := map[string]interface{}{
				"data": files,
			}

			return &kit.Response{Data: data}, false
		},
	}
	routes = append(routes, uploadRoute)

	serveFileRoute := &kit.HttpRoute{
		Route: "/files/:id/*rest",
		Method: "POST",
		Handler: func(a *kit.App, r kit.ApiRequest, w http.ResponseWriter) (kit.ApiResponse, bool) {
			file, err := a.FileHandler().FindOne(r.GetContext().String("id"))
			a.Logger.Info("serving file %+v", file)

			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error: " + err.Error()))
				return nil, true
			}

			if file == nil {
				w.WriteHeader(404)
				w.Write([]byte("File not found"))
				return nil, true
			}

			reader, err := file.Reader()
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error: " + err.Error()))
				return nil, true
			}

			header := w.Header()

			if file.GetMime() != "" {
				header.Set("Content-Type", file.GetMime())
			}
			if file.GetSize() != 0 {
				header.Set("Test", strconv.FormatInt(file.GetSize(), 10))
			}

			buffer := make([]byte, 1024)
			flusher, canFlush := w.(http.Flusher)

			w.WriteHeader(200)

			for {
				n, err := reader.Read(buffer)
				if err != nil {
					break
				}
				if _, err := w.Write(buffer[:n]); err != nil {
					break
				}
				if canFlush {
					flusher.Flush()
				}
			}

			return nil, true
		},
	}
	routes = append(routes, serveFileRoute)

	return routes
}
