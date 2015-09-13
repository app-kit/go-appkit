package files

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/julienschmidt/httprouter"
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

	tmpFile := r.GetMeta().GetString("file")
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

func (res FilesResource) handleUpload(tmpPath string, r *http.Request) ([]string, kit.ApiError) {
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

func (hooks FilesResource) HttpRoutes(res kit.ApiResource, router *httprouter.Router) {
	router.OPTIONS("/api/files/upload", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		header := w.Header()

		allowedOrigins := res.App().Config.UString("fileHandler.allowedOrigins", "*")
		header.Set("Access-Control-Allow-Origin", allowedOrigins)

		header.Set("Access-Control-Allow-Methods", "OPTIONS, POST")
		header.Set("Access-Control-Allow-Headers", "Authentication, Content-Type, Content-Range, Content-Disposition")

		w.WriteHeader(200)
	})

	tmpPath := getTmpPath(res)
	if tmpPath == "" {
		panic("Empty tmp path")
	}

	// Route for uploading files.
	router.POST("/api/files/upload", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		var data map[string]interface{}
		code := 200

		var err kit.ApiError = nil

		if res.App().Config.UBool("fileHandler.requiresAuth", false) {
			// Authentication is required, so authenticate user first.
			authHeaderName := res.App().Config.UString("authHeader", "Authentication")
			token := r.Header.Get(authHeaderName)

			if token == "" {
				err = kit.Error{
					Code:    "auth_header_missing",
					Message: "Authentication is required, but Authentication header is missing",
				}
				code = 403
			}

			_, session, err := res.GetUserHandler().VerifySession(token)
			if err != nil {
				err = kit.Error{Code: "auth_error"}
				code = 500
			}
			if session == nil {
				err = kit.Error{Code: "invalid_token"}
				code = 403
			}
		}

		var files []string

		if err == nil {
			files, err = hooks.handleUpload(tmpPath, r)
			if err != nil {
				code = 500
			}
		}

		if err != nil {
			data = map[string]interface{}{
				"errors": []error{err},
			}
		} else {
			data = map[string]interface{}{
				"data": files,
			}
		}

		json, err2 := json.Marshal(data)
		if err2 != nil {
			json = []byte(`{"errors": [{code: "json_marshal_failed"}]}`)
		}

		w.WriteHeader(code)
		w.Write(json)
	})

	router.GET("/files/:id/*rest", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		file, err := res.App().FileHandler().FindOne(params.ByName("id"))

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error: " + err.Error()))
			return
		}

		if file == nil {
			w.WriteHeader(404)
			w.Write([]byte("File not found"))
		}

		reader, err := file.Reader()
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error: " + err.Error()))
			return
		}

		header := w.Header()

		if file.GetMime() != "" {
			header.Set("Content-Type", file.GetMime())
		}
		if file.GetSize() != 0 {
			header.Set("Test", strconv.FormatInt(file.GetSize(), 10))
		}

		buffer := make([]byte, 1024)
		flusher, canFlush := res.(http.Flusher)

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
	})
}
