package files

import(
	"net/http"
	"os"
	"io"
	"encoding/json"

	"github.com/julienschmidt/httprouter"
	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
)

type FilesResource struct {
}


func (res FilesResource) handleUpload(tmpPath string, r *http.Request) ([]string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, kit.Error{
					Code: "read_error",
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
				Code: "create_dir_failed",
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
				Code: "file_create_failed",
				Message: err.Error(),
			}
		}

		_, err = io.Copy(file, part)
		if err != nil {
			return nil, kit.Error{
				Code: "file_create_failed",
				Message: err.Error(),
			}
		}

		files = append(files, id + string(os.PathSeparator) + filename)
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

	tmpPath := res.App().Config.UString("tmpDir")
	if tmpPath == "" {
		panic("Empty tmp path")
	}

	router.POST("/api/files/upload", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		files, err := hooks.handleUpload(tmpPath, r)

		var data map[string]interface{}
		code := 200

		if err != nil {
			data = map[string]interface{}{
				"errors": []error{err},
			}
			code = 500
		} else {
			data = map[string]interface{}{
				"data": files,
			}
		}

		json, err := json.Marshal(data)
		if err != nil {
			json = []byte(`{"errors": [{code: "json_marshal_failed"}]}`)
		}

		w.WriteHeader(code)
		w.Write(json)
	})
}
