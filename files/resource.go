package files

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"github.com/disintegration/gift"
	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/utils"
	. "github.com/theduke/go-appkit/error"
	db "github.com/theduke/go-dukedb"
)

type rateLimiter struct {
	sync.Mutex

	running           int
	maxPerIPPerMinute int
	maxQueueSize      int

	maxRunning    int
	ipLog         map[string][]*time.Time
	queueChannels []chan bool
}

func newRateLimiter(maxRunning, maxPerIPPerMinute int, maxQueueSize int) *rateLimiter {
	limiter := &rateLimiter{
		maxRunning:        maxRunning,
		maxPerIPPerMinute: maxPerIPPerMinute,
		maxQueueSize:      maxQueueSize,

		ipLog:         make(map[string][]*time.Time),
		queueChannels: make([]chan bool, 0),
	}

	return limiter
}

func (r *rateLimiter) PruneIpLog() {
	now := time.Now()

	for ip := range r.ipLog {
		for index, t := range r.ipLog[ip] {
			if now.Sub(*t).Seconds() > 60 {
				if index == len(r.ipLog[ip])-1 {
					r.ipLog[ip] = r.ipLog[ip][:]
				} else {
					r.ipLog[ip] = r.ipLog[ip][index+1:]
				}
				break
			}
		}

		if len(r.ipLog[ip]) == 0 {
			delete(r.ipLog, ip)
		}
	}
}

func (r *rateLimiter) Start(ip string) (chan bool, Error) {
	if r.running >= r.maxRunning {
		if len(r.queueChannels) >= r.maxQueueSize {
			return nil, AppError{
				Code:    "rate_limit_queue_threshold_exceeded",
				Message: "The queue for the rate limiter has reached it's maximum size",
			}
		} else {
			channel := make(chan bool)
			r.Lock()
			r.queueChannels = append(r.queueChannels, channel)
			r.Unlock()
			return channel, nil
		}
	}

	// Check for ip limits.
	r.PruneIpLog()
	if log, ok := r.ipLog[ip]; ok {
		if len(log) > r.maxPerIPPerMinute {
			return nil, AppError{
				Code:    "rate_limit_max_per_ip_per_minute_exceeced",
				Message: "The maximum limit for requests per ip per minute was exceeded",
			}
		}
	}

	// maxRunning is not reached, so it is allowed to start.
	r.Lock()
	if _, ok := r.ipLog[ip]; !ok {
		r.ipLog[ip] = make([]*time.Time, 0)
	}
	r.running += 1
	now := time.Now()
	r.ipLog[ip] = append(r.ipLog[ip], &now)
	r.Unlock()

	return nil, nil
}

func (r *rateLimiter) Finish() {
	var channel chan bool
	r.Lock()
	r.running -= 1
	if len(r.queueChannels) > 0 {
		channel = r.queueChannels[0]
		r.queueChannels = r.queueChannels[1:]
	}
	r.Unlock()

	if channel != nil {
		channel <- true
	}
}

type FilesResource struct {
	thumbnailRateLimiter *rateLimiter
}

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
			Error: AppError{
				Code:    "no_tmp_path",
				Message: "Tmp path is not configured",
			},
		}
	}

	tmpFile := r.GetMeta().String("file")
	if tmpFile == "" {
		return &kit.Response{
			Error: AppError{
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

func handleUpload(a *kit.App, tmpPath string, r *http.Request) ([]string, Error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, AppError{Code: "multipart_error", Message: err.Error()}
	}

	files := make([]string, 0)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, AppError{
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
			return nil, AppError{
				Code:    "create_dir_failed",
				Message: err.Error(),
			}
		}

		filename = utils.Canonicalize(filename)
		if filename == "" {
			filename = id
		}

		filePath := path + string(os.PathSeparator) + filename

		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return nil, AppError{
				Code:    "file_create_failed",
				Message: err.Error(),
			}
		}
		defer file.Close()

		_, err = io.Copy(file, part)
		if err != nil {
			return nil, AppError{
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

func (r *FilesResource) getImageReader(app *kit.App, tmpDir string, file kit.ApiFile, width, height int64, ip string) (io.Reader, Error) {
	if width == 0 && height == 0 {
		return file.Reader()
	}

	// Dimensions specified.
	// Check if the thumbnail was already created.
	// If so, serve it. Otherwise, create it first.

	if (width == 0 || height == 0) && (file.GetWidth() == 0 || file.GetHeight() == 0) {
		return nil, AppError{
			Code:     "image_dimensions_not_determined",
			Message:  fmt.Sprintf("The file with id %v does not have width/height", file.GetID()),
			Internal: true,
		}
	}

	if width < 0 || height < 0 {
		return nil, AppError{
			Code: "invalid_dimensions",
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

	maxWidth := app.Config.UInt("files.thumbGenerator.maxWidth", 2000)
	maxHeight := app.Config.UInt("files.thumbGenerator.maxHeight", 2000)

	if width > int64(maxWidth) || height > int64(maxHeight) {
		return nil, AppError{
			Code:    "dimensions_exceed_maximum_limits",
			Message: "The specified dimensions exceed the maximum limits",
		}
	}

	thumbId := fmt.Sprintf("%v_%v_%v_%v_%v.%v",
		file.GetID(),
		file.GetBucket(),
		file.GetName(),
		strconv.FormatInt(width, 10),
		strconv.FormatInt(height, 10),
		"jpeg")

	if ok, _ := file.GetBackend().HasFileById("thumbs", thumbId); !ok {
		channel, err := r.thumbnailRateLimiter.Start(ip)
		if err != nil {
			return nil, err
		}
		if channel != nil {
			<-channel
		}

		// Thumb does not exist yet, so create it.
		reader, err := file.Reader()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		img, _, err2 := image.Decode(reader)
		if err2 != nil {
			return nil, AppError{
				Code:    "image_decode_error",
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

		r.thumbnailRateLimiter.Finish()
	}

	return file.GetBackend().ReaderById("thumbs", thumbId)
}

func (hooks FilesResource) HttpRoutes(res kit.ApiResource) []*kit.HttpRoute {
	maxRunning := res.App().Config.UInt("files.thumbGenerator.maxRunning", 10)
	maxPerIPPerMinute := res.App().Config.UInt("files.thumbGenerator.maxPerIPPerMinute", 100)
	maxQueueSize := res.App().Config.UInt("files.thumbGenerator.maxQueueSize", 100)
	hooks.thumbnailRateLimiter = newRateLimiter(maxRunning, maxPerIPPerMinute, maxQueueSize)

	routes := make([]*kit.HttpRoute, 0)

	// Upload route.
	uploadOptionsRoute := &kit.HttpRoute{
		Route:  "/api/files/upload",
		Method: "OPTIONS",
		Handler: func(a *kit.App, r kit.ApiRequest) (kit.ApiResponse, bool) {
			header := r.GetContext().MustGet("ResponseWriter").(http.ResponseWriter).Header()

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

			return &kit.Response{
				HttpStatus: 200,
				RawData: []byte{},
			}, true
		},
	}
	routes = append(routes, uploadOptionsRoute)

	tmpPath := getTmpPath(res)
	if tmpPath == "" {
		panic("Empty tmp path")
	}

	uploadRoute := &kit.HttpRoute{
		Route:  "/api/files/upload",
		Method: "POST",
		Handler: func(a *kit.App, r kit.ApiRequest) (kit.ApiResponse, bool) {
			if a.Config.UBool("fileHandler.requiresAuth", false) {
				if r.GetUser() == nil {
					return kit.NewErrorResponse("permission_denied", ""), false
				}
			}

			var files []string
			var err Error

			if err == nil {
				files, err = handleUpload(a, tmpPath, r.GetContext().MustGet("httpRequest").(*http.Request))
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
		Route:  "/files/:id/*rest",
		Method: "GET",
		Handler: func(a *kit.App, r kit.ApiRequest) (kit.ApiResponse, bool) {
			file, err := a.FileHandler().FindOne(r.GetContext().String("id"))

			if err != nil {
				return &kit.Response{
					Error: err,
				}, false
			}

			if file == nil {
				return &kit.Response{
					HttpStatus: 404,
					RawData: []byte("File not found"),
				}, false
			}

			reader, err := file.Reader()
			if err != nil {
				return &kit.Response{
					Error: err,
				}, false
			}
			defer reader.Close()

			w := r.GetContext().MustGet("responseWriter").(http.ResponseWriter)
			serveFile(w, file, reader)
			return nil, true
		},
	}
	routes = append(routes, serveFileRoute)

	serveImageRoute := &kit.HttpRoute{
		Route:  "/images/:id/*rest",
		Method: "GET",
		Handler: func(a *kit.App, r kit.ApiRequest) (kit.ApiResponse, bool) {
			file, err := a.FileHandler().FindOne(r.GetContext().String("id"))
			if err != nil {
				return &kit.Response{
					Error: err,
				}, false
			}

			if file == nil {
				return &kit.Response{
					HttpStatus: 404,
					RawData: []byte("File not found"),
				}, false
			}

			if !file.GetIsImage() {
				return &kit.Response{
					Error: AppError{
						Code: "file_is_no_image",
						Message: "The requested file is not an image",
					},
				}, false
			}

			httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)

			query := httpRequest.URL.Query()
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

			ip := strings.Split(httpRequest.RemoteAddr, ":")[0]
			if ip == "" {
				ip = httpRequest.Header.Get("X-Forwarded-For")
			}

			reader, err := hooks.getImageReader(a, thumbDir, file, width, height, ip)
			if err != nil {
				return &kit.Response{
					Error: err,
				}, false
			}

			w := r.GetContext().MustGet("responseWriter").(http.ResponseWriter)
			serveFile(w, file, reader)

			return nil, true
		},
	}
	routes = append(routes, serveImageRoute)

	return routes
}
