package files

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"fmt"
	"image"

	_ "image/png"
	"image/jpeg"
	_ "image/gif"

	"github.com/twinj/uuid"
	"github.com/disintegration/gift"

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
		defer file.Close()

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

func serveFile(w http.ResponseWriter, file kit.ApiFile, reader io.Reader) {
	header := w.Header()

	if file.GetMime() != "" {
		header.Set("Content-Type", file.GetMime())
	}
	/*
	if file.GetSize() != 0 {
		header.Set("Content-Length", strconv.FormatInt(file.GetSize(), 10))
	}
	*/

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
}

func getImageReader(tmpDir string, file kit.ApiFile, width, height int64) (io.Reader, kit.ApiError) {
	if width == 0 && height == 0 {
		return file.Reader()
	}

	// Dimensions specified.
	// Check if the thumbnail was already created.
	// If so, serve it. Otherwise, create it first.

	if (width == 0 || height == 0) && (file.GetWidth() == 0 || file.GetHeight() == 0) {
		return nil, kit.Error{
			Code: "image_dimensions_not_determined",
			Message: fmt.Sprintf("The file with id %v does not have width/height", file.GetID()),
			Internal: true,
		}
	}

	// If either height or width is 0, determine proper values to presserve aspect ratio.
  if width == 0 {
  	ratio := float64(file.GetWidth()) / float64(file.GetHeight())
  	width = int64(float64(height) * ratio)
  } else if height == 0 {
  	ratio := float64(file.GetHeight()) / float64(file.GetWidth())
  	height = int64(float64(width) * ratio)
  }

	thumbId := fmt.Sprintf("%v_%v_%v_%v_%v.%v",
		file.GetID(),
		file.GetBucket(),
		file.GetName(),
		strconv.FormatInt(width, 10),
		strconv.FormatInt(height, 10),
		"jpeg")

	if ok, _ := file.GetBackend().HasFileById("thumbs", thumbId); !ok {
		// Thumb does not exist yet, so create it.
		reader, err := file.Reader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

    img, _, err2 := image.Decode(reader)
    if err2 != nil {
    	return nil, kit.Error{
				Code: "image_decode_error",
				Message: err2.Error(),
			}
    }

    gift := gift.New(
	    gift.ResizeToFill(int(width), int(height), gift.LanczosResampling, gift.CenterAnchor),
		)
		thumb := image.NewRGBA(gift.Bounds(img.Bounds()))
		gift.Draw(thumb, img)

    _, writer, err := file.GetBackend().WriterById("thumbs", thumbId, true)
    if err != nil {
    	return nil, err
    }
    defer writer.Close()

    jpeg.Encode(writer, thumb, &jpeg.Options{Quality: 90})
	}

	return file.GetBackend().ReaderById("thumbs", thumbId)
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
		Method: "GET",
		Handler: func(a *kit.App, r kit.ApiRequest, w http.ResponseWriter) (kit.ApiResponse, bool) {
			file, err := a.FileHandler().FindOne(r.GetContext().String("id"))
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
			defer reader.Close()

			serveFile(w, file, reader)
			return nil, true
		},
	}
	routes = append(routes, serveFileRoute)

	serveImageRoute := &kit.HttpRoute{
		Route: "/images/:id/*rest",
		Method: "GET",
		Handler: func(a *kit.App, r kit.ApiRequest, w http.ResponseWriter) (kit.ApiResponse, bool) {
			file, err := a.FileHandler().FindOne(r.GetContext().String("id"))
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error: " + err.Error()))
				return nil, true
			}

			if file == nil {
				w.WriteHeader(404)
				w.Write([]byte("Image not found"))
				return nil, true
			}

			if !file.GetIsImage() {
				w.WriteHeader(404)
				w.Write([]byte("Image not found"))
				return nil, true
			}

			query := r.GetHttpRequest().URL.Query()
			rawWidth := query.Get("width")
			rawHeight := query.Get("height")

			var width, height int64

			if rawWidth != "" {
				width, _ = strconv.ParseInt(rawWidth, 10, 64)
			}
			if rawHeight != "" {
				height, _ = strconv.ParseInt(rawHeight, 10, 64)
			}

			thumbDir := a.Config.UString("thumbnailDir")
			if thumbDir == "" {
				thumbDir = tmpPath + string(os.PathSeparator) + "thumbnails"
			}

			reader, err := getImageReader(thumbDir, file, width, height)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Error: " + err.Error()))
				return nil, true
			}
			serveFile(w, file, reader)

			return nil, true
		},
	}
	routes = append(routes, serveImageRoute)

	return routes
}
