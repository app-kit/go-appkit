package files

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"github.com/disintegration/gift"
	"github.com/theduke/go-apperror"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/utils"
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

func (r *rateLimiter) Start(ip string) (chan bool, apperror.Error) {
	if r.running >= r.maxRunning {
		if len(r.queueChannels) >= r.maxQueueSize {
			return nil, &apperror.Err{
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
			return nil, &apperror.Err{
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

func getTmpPath(res kit.Resource) string {
	c := res.Registry().Config()
	tmpPath := c.UPath("tmpUploadDir")
	if tmpPath == "" {
		tmpPath = c.TmpDir()
		if tmpPath == "" {
			panic("config.TmpDir() empty")
		}
		tmpPath = path.Join(tmpPath, "uploads")
	}

	return tmpPath
}

func (_ FilesResource) ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response {
	// Verify that tmp path is set either in metadata or on model.

	file := obj.(kit.File)
	if file.GetTmpPath() == "" {
		file.SetTmpPath(r.GetMeta().String("file"))
	}

	filePath := file.GetTmpPath()
	if filePath == "" {
		return kit.NewErrorResponse("no_tmp_path", "A tmp path must be set when creating a file", true)
	}

	tmpPath := getTmpPath(res)

	if !strings.HasPrefix(filePath, tmpPath) && filePath[0] != '/' {
		filePath = tmpPath + string(os.PathSeparator) + filePath
		file.SetTmpPath(filePath)
	}

	// Build the file, save it to backend and persist it to the db.
	err := res.Registry().FileService().BuildFile(file, r.GetUser(), true, true)
	if err != nil {
		kit.NewErrorResponse(err)
	}

	return &kit.AppResponse{
		Data: file,
	}
}

func handleUpload(registry kit.Registry, tmpPath string, r *http.Request) ([]string, apperror.Error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, apperror.Wrap(err, "multipart_error")
	}

	files := make([]string, 0)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, apperror.Wrap(err, "read_error")
			}
		}

		filename := part.FileName()
		if filename == "" {
			// Not a file?
			continue
		}

		id := utils.UUIDv4()
		path := tmpPath + string(os.PathSeparator) + id

		if err := os.MkdirAll(path, 0777); err != nil {
			return nil, apperror.Wrap(err, "create_dir_failed")
		}

		filename = utils.Canonicalize(filename)
		if filename == "" {
			filename = id
		}

		filePath := path + string(os.PathSeparator) + filename

		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return nil, apperror.Wrap(err, "file_create_failed")
		}
		defer file.Close()

		_, err = io.Copy(file, part)
		if err != nil {
			return nil, apperror.Wrap(err, "file_copy_failed")
		}

		files = append(files, id+string(os.PathSeparator)+filename)
	}

	return files, nil
}

func parseRangeHeader(header string) (int64, int64, apperror.Error) {
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, apperror.New("invalid_range_header")
	}
	parts := strings.Split(header[6:], "-")
	if len(parts) != 2 {
		return 0, 0, apperror.New("invalid_range_header")
	}

	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, apperror.New("invalid_range_start")
	}

	end := int64(-1)
	if parts[1] != "" {
		x, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, apperror.New("invalid_range_end")
		}
		end = x
	}

	return start, end, nil
}

func serveFile(w http.ResponseWriter, request *http.Request, mime string, size int64, reader io.ReadSeeker) apperror.Error {
	header := w.Header()

	if mime != "" {
		header.Set("Content-Type", mime)
	}

	if size != 0 {
		header.Set("Content-Length", strconv.FormatInt(size, 10))
	}

	header.Set("Accept-Ranges", "bytes")

	start := int64(0)
	end := size

	// Handle range requests.
	rangeHeader := request.Header.Get("Range")
	if rangeHeader != "" {
		var err error
		start, end, err = parseRangeHeader(rangeHeader)
		if err != nil {
			w.WriteHeader(416)
			return apperror.Wrap(err, "file_serve_invalid_range_header")
		}
		if end == -1 {
			end = size
		}
	}

	fmt.Printf("Header: %+v\n", request.Header)
	fmt.Printf("range: %v | start: %v | end %v\n", rangeHeader, start, end)

	if start > 0 {
		if _, err := reader.Seek(start, 0); err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Internal server error."))

			return apperror.Wrap(err, "file_serve_seek_error")
		}

		// Write content range related headers.
		contentRange := fmt.Sprintf("bytes %v-%v/%v", start, end, size)
		fmt.Printf("writing content-range header: %v\n", contentRange)
		header.Set("Content-Range", contentRange)
		w.WriteHeader(206)
	} else {
		w.WriteHeader(200)
	}

	buffer := make([]byte, 1024)
	flusher, canFlush := w.(http.Flusher)

	pos := start

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return apperror.Wrap(err, "file_server_read_error")
		}

		if pos+int64(n) > end {
			fmt.Printf("Reached end of range: pos %v, read: %v, end: %v\n", pos, n, end)
			n = int(end - pos)
			fmt.Printf("Reduced n to %v\n", n)
		}

		pos += int64(n)

		if n > 0 {
			if _, err2 := w.Write(buffer[:n]); err2 != nil {
				fmt.Printf("Tried to serve %v bytes from position %v\n", n, pos)
				return apperror.Wrap(err, "file_serve_write_error", fmt.Sprintf("Error while serving %v bytes as position %v", n, pos))
			}

			if canFlush {
				flusher.Flush()
			}
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

func (r *FilesResource) getImageReader(registry kit.Registry, tmpDir string, file kit.File, width, height int64, filters []string, ip string) (reader kit.ReadSeekerCloser, size int64, err apperror.Error) {
	if width == 0 && height == 0 && len(filters) == 0 {
		reader, err = file.Reader()
		return
	}

	// Dimensions specified.
	// Check if the thumbnail was already created.
	// If so, serve it. Otherwise, create it first.

	if (width == 0 || height == 0) && (file.GetWidth() == 0 || file.GetHeight() == 0) {
		err = &apperror.Err{
			Code:    "image_dimensions_not_determined",
			Message: fmt.Sprintf("The file with id %v does not have width/height", file.GetID()),
		}
		return
	}

	if width < 0 || height < 0 {
		err = apperror.New("invalid_dimensions")
		return
	}

	// If either height or width is 0, determine proper values to presserve aspect ratio.
	if width == 0 {
		ratio := float64(file.GetWidth()) / float64(file.GetHeight())
		width = int64(float64(height) * ratio)
	} else if height == 0 {
		ratio := float64(file.GetHeight()) / float64(file.GetWidth())
		height = int64(float64(width) * ratio)
	}

	maxWidth := registry.Config().UInt("files.thumbGenerator.maxWidth", 2000)
	maxHeight := registry.Config().UInt("files.thumbGenerator.maxHeight", 2000)

	if width > int64(maxWidth) || height > int64(maxHeight) {
		err = &apperror.Err{
			Code:    "dimensions_exceed_maximum_limits",
			Message: "The specified dimensions exceed the maximum limits",
		}
		return
	}

	thumbId := fmt.Sprintf("%v_%v_%v_%v_%v_%v.%v",
		file.GetID(),
		file.GetBucket(),
		file.GetName(),
		strconv.FormatInt(width, 10),
		strconv.FormatInt(height, 10),
		strings.Replace(strings.Join(filters, "_"), ":", "_", -1),
		"jpeg")

	if ok, _ := file.GetBackend().HasFileById("thumbs", thumbId); !ok {
		var channel chan bool
		channel, err = r.thumbnailRateLimiter.Start(ip)
		if err != nil {
			return
		}
		if channel != nil {
			<-channel
		}

		// Thumb does not exist yet, so create it.
		reader, err = file.Reader()
		if err != nil {
			return
		}
		defer reader.Close()

		img, _, err2 := image.Decode(reader)
		if err2 != nil {
			err = apperror.Wrap(err2, "image_decode_error")
			return
		}

		var giftFilters []gift.Filter

		if !(height == 0 && width == 0) {
			giftFilters = append(giftFilters, gift.ResizeToFill(int(width), int(height), gift.LanczosResampling, gift.CenterAnchor))
		}

		for _, filter := range filters {
			if filter == "" {
				continue
			}

			parts := strings.Split(filter, ":")

			if len(parts) > 1 {
				filter = parts[0]
			}

			switch filter {
			case "sepia":
				n := float32(100)

				if len(parts) == 2 {
					x, err2 := strconv.ParseFloat(parts[1], 64)
					if err2 == nil {
						n = float32(x)
					} else {
						err = apperror.New("invalid_sepia_filter_value", true)
						return
					}
				}

				giftFilters = append(giftFilters, gift.Sepia(n))

			case "grayscale":
				giftFilters = append(giftFilters, gift.Grayscale())

			case "brightness":
				n := float32(0)

				if len(parts) == 2 {
					x, err2 := strconv.ParseFloat(parts[1], 64)
					if err2 == nil {
						n = float32(x)
					} else {
						err = apperror.New("invalid_brightness_filter_value", true)
						return
					}
				}

				giftFilters = append(giftFilters, gift.Brightness(n))

			default:
				err = apperror.New("unknown_filter", fmt.Sprintf("Unknown filter: %v", filter), true)
				return
			}
		}

		gift := gift.New(giftFilters...)

		thumb := image.NewRGBA(gift.Bounds(img.Bounds()))
		gift.Draw(thumb, img)

		var writer io.WriteCloser
		_, writer, err = file.GetBackend().WriterById("thumbs", thumbId, true)
		if err != nil {
			return
		}
		defer writer.Close()

		jpeg.Encode(writer, thumb, &jpeg.Options{Quality: 90})

		r.thumbnailRateLimiter.Finish()
	}

	backend := file.GetBackend()
	size, err = backend.FileSizeById("thumbs", thumbId)
	if err != nil {
		return
	}

	reader, err = file.GetBackend().ReaderById("thumbs", thumbId)
	return
}

func (hooks FilesResource) HttpRoutes(res kit.Resource) []kit.HttpRoute {
	maxRunning := res.Registry().Config().UInt("files.thumbGenerator.maxRunning", 10)
	maxPerIPPerMinute := res.Registry().Config().UInt("files.thumbGenerator.maxPerIPPerMinute", 100)
	maxQueueSize := res.Registry().Config().UInt("files.thumbGenerator.maxQueueSize", 100)
	hooks.thumbnailRateLimiter = newRateLimiter(maxRunning, maxPerIPPerMinute, maxQueueSize)

	routes := make([]kit.HttpRoute, 0)

	// Upload route.
	uploadOptionsHandler := func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		header := r.GetHttpResponseWriter().Header()

		allowedOrigins := registry.Config().UString("fileHandler.allowedOrigins", "*")
		header.Set("Access-Control-Allow-Origin", allowedOrigins)

		header.Set("Access-Control-Allow-Methods", "OPTIONS, POST")

		allowedHeaders := registry.Config().UString("accessControl.allowedHeaders")
		if allowedHeaders == "" {
			allowedHeaders = "Authentication, Content-Type, Content-Range, Content-Disposition"
		} else {
			allowedHeaders += ", Authentication, Content-Type, Content-Range, Content-Disposition"
		}
		header.Set("Access-Control-Allow-Headers", allowedHeaders)

		return &kit.AppResponse{
			HttpStatus: 200,
			RawData:    []byte{},
		}, true
	}

	uploadOptionsRoute := kit.NewHttpRoute("/api/file-upload", "OPTIONS", uploadOptionsHandler)
	routes = append(routes, uploadOptionsRoute)

	tmpPath := getTmpPath(res)
	if tmpPath == "" {
		panic("Empty tmp path")
	}

	uploadHandler := func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		if registry.Config().UBool("fileHandler.requiresAuth", false) {
			if r.GetUser() == nil {
				return kit.NewErrorResponse("permission_denied", ""), false
			}
		}

		var files []string
		var err apperror.Error

		if err == nil {
			files, err = handleUpload(registry, tmpPath, r.GetHttpRequest())
			if err != nil {
				return kit.NewErrorResponse(err), false
			}
		}

		data := map[string]interface{}{
			"data": files,
		}

		return &kit.AppResponse{Data: data}, false
	}
	uploadRoute := kit.NewHttpRoute("/api/file-upload", "POST", uploadHandler)
	routes = append(routes, uploadRoute)

	serveFileHandler := func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		file, err := registry.FileService().FindOne(r.GetContext().String("id"))

		if err != nil {
			return kit.NewErrorResponse(err), false
		}

		if file == nil {
			return &kit.AppResponse{
				HttpStatus: 404,
				RawData:    []byte("File not found"),
			}, false
		}

		reader, err := file.Reader()
		if err != nil {
			return kit.NewErrorResponse(err), false
		}
		defer reader.Close()

		w := r.GetHttpResponseWriter()

		err = serveFile(w, r.GetHttpRequest(), file.GetMime(), file.GetSize(), reader)
		reader.Close()

		if err != nil {
			registry.Logger().Errorf("Error while serving file %v(%v): %v", file.GetID(), file.GetBackendID(), err)
		}

		return nil, true
	}
	serveFileRoute := kit.NewHttpRoute("/files/:id/*rest", "GET", serveFileHandler)
	routes = append(routes, serveFileRoute)

	serveImageHandler := func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		file, err := registry.FileService().FindOne(r.GetContext().String("id"))
		if err != nil {
			return kit.NewErrorResponse(err), false
		}

		if file == nil {
			return &kit.AppResponse{
				HttpStatus: 404,
				RawData:    []byte("File not found"),
			}, false
		}

		if !file.GetIsImage() {
			return &kit.AppResponse{
				Error: &apperror.Err{
					Code:    "file_is_no_image",
					Message: "The requested file is not an image",
				},
			}, false
		}

		httpRequest := r.GetHttpRequest()

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

		rawFilters := query.Get("filters")
		filters := strings.Split(rawFilters, ",")

		thumbDir := registry.Config().UString("thumbnailDir")
		if thumbDir == "" {
			thumbDir = tmpPath + string(os.PathSeparator) + "thumbnails"
		}

		ip := strings.Split(httpRequest.RemoteAddr, ":")[0]
		if ip == "" {
			ip = httpRequest.Header.Get("X-Forwarded-For")
		}

		reader, size, err := hooks.getImageReader(registry, thumbDir, file, width, height, filters, ip)
		if err != nil {
			return kit.NewErrorResponse(err), false
		}

		w := r.GetHttpResponseWriter()

		err = serveFile(w, httpRequest, file.GetMime(), size, reader)
		reader.Close()

		if err != nil {
			registry.Logger().Errorf("Error while serving image %v(%v): %v", file.GetID(), file.GetBackendID(), err)
		}

		return nil, true
	}
	serveImageRoute := kit.NewHttpRoute("/images/:id/*rest", "GET", serveImageHandler)
	routes = append(routes, serveImageRoute)

	return routes
}
