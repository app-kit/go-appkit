package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/caches"
	"github.com/theduke/go-appkit/utils"
)

type HttpHandlerStruct struct {
	App     kit.App
	Handler kit.RequestHandler
}

func (h *HttpHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := new(httprouter.Params)
	httpHandler(w, r, *params, h.App, h.Handler)
}

func serverRenderer(app kit.App, r kit.Request) kit.Response {
	url := r.GetContext().MustGet("httpRequest").(*http.Request).URL

	// Build the url to query.
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	if url.Host == "" {
		url.Host = app.Config().UString("host", "localhost") + ":" + app.Config().UString("port", "8000")
	}

	q := url.Query()
	q.Set("no-server-render", "1")
	url.RawQuery = q.Encode()

	strUrl := url.String()

	cacheKey := "serverrenderer_" + strUrl
	cacheName := app.Config().UString("serverRenderer.cache")
	var cache kit.Cache

	// If a cache is specified, try to retrieve it.
	if cacheName != "" {
		cache = app.Cache(cacheName)
		if cache == nil {
			app.Logger().Errorf("serverRenderer.cache is set to %v, but the cache is not registered with app", cacheName)
		}
	}

	// If a cache was found, try to retrieve cached response.
	if cache != nil {
		item, err := cache.Get(cacheKey)
		if err != nil {
			app.Logger().Errorf("serverRenderer: cache retrieval error: %v", err)
		} else if item != nil {
			// Cache item found, return response with cache item.
			status, _ := strconv.ParseInt(item.GetTags()[0], 10, 64)
			data, _ := item.ToString()

			return &kit.AppResponse{
				HttpStatus: int(status),
				RawData:    []byte(data),
			}
		}
	}

	// Either no cache or url not yet cached, so render it.

	// First, ensure that the tmp directory exists.
	tmpDir := path.Join(app.TmpDir(), "phantom")
	if ok, _ := utils.FileExists(tmpDir); !ok {
		if err := os.MkdirAll(tmpDir, 0777); err != nil {
			return &kit.AppResponse{
				Error: kit.AppError{
					Code:     "create_tmp_dir_failed",
					Message:  fmt.Sprintf("Could not create the tmp directory at %v: %v", tmpDir, err),
					Internal: true,
				},
			}
		}
	}

	// Build a unique file name.
	filePath := path.Join(tmpDir, uuid.NewV4().String()+".html")

	// Execute phantom js.

	// Find path of phantom script.
	_, filename, _, _ := runtime.Caller(1)
	scriptPath := path.Join(path.Dir(path.Dir(filename)), "phantom", "render.js")

	start := time.Now()

	phantomPath := app.Config().UString("serverRenderer.phantomJsPath", "phantomjs")

	args := []string{
		"--web-security=false",
		"--local-to-remote-url-access=true",
		scriptPath,
		"10",
		strUrl,
		filePath,
	}
	result, err := exec.Command(phantomPath, args...).CombinedOutput()
	if err != nil {
		app.Logger().Errorf("Phantomjs execution error: %v", string(result))

		return &kit.AppResponse{
			Error: kit.AppError{
				Code:     "phantom_execution_failed",
				Message:  err.Error(),
				Data:     result,
				Errors:   []error{err},
				Internal: true,
			},
		}
	}

	// Get time taken as milliseconds.
	timeTaken := int(time.Now().Sub(start) / time.Millisecond)
	app.Logger().WithFields(log.Fields{
		"action":       "phantomjs_render",
		"milliseconds": timeTaken,
	}).Debugf("Rendered url %v with phantomjs", url)

	content, err2 := utils.ReadFile(filePath)
	if err2 != nil {
		return &kit.AppResponse{Error: err2}
	}

	// Find http status code.
	status := 200
	res := regexp.MustCompile("http_status_code\\=(\\d+)").FindStringSubmatch(string(content))
	if res != nil {
		s, _ := strconv.ParseInt(res[1], 10, 64)
		status = int(s)
	}

	// Save to cache.
	if cache != nil {
		lifetime := app.Config().UInt("serverRenderer.cacheLiftetime", 3600)

		err := cache.Set(&caches.StrItem{
			Key:       cacheKey,
			Value:     string(content),
			Tags:      []string{strconv.FormatInt(int64(status), 10)},
			ExpiresAt: time.Now().Add(time.Duration(lifetime) * time.Second),
		})
		if err != nil {
			app.Logger().Errorf("serverRenderer: Cache persist error: %v", err)
		}
	}

	return &kit.AppResponse{
		HttpStatus: status,
		RawData:    content,
	}
}

func defaultErrorTpl() *template.Template {
	tpl := `
	<html>
		<head>
			<title>Server Error</title>
		</head>

		<body>
			<h1>Server Error</h1>

			<p>{{.error}}</p>
		</body>
	</html>
	`

	t := template.Must(template.New("error").Parse(tpl))
	return t
}

func defaultNotFoundTpl() *template.Template {
	tpl := `
	<html>
		<head>
			<title>Page Not Found</title>
		</head>

		<body>
			<h1>Page Not Found</h1>

			<p>The page you are looking for does not exist.</p>
		</body>
	</html>
	`

	t, _ := template.New("error").Parse(tpl)
	return t
}

func getIndexTpl(app kit.App) ([]byte, kit.Error) {
	if path := app.Config().UString("frontend.indexTpl"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, kit.AppError{
				Code:    "cant_open_index_tpl",
				Message: fmt.Sprintf("The index template at %v could not be opened: %v", path, err),
			}
		}

		tpl, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, kit.AppError{
				Code:    "index_tpl_read_error",
				Message: fmt.Sprintf("Could not read index template at %v: %v", path, err),
			}
		}

		return tpl, nil
	}

	tpl := `
	<html>
		<body>
			<h1>Go Appkit</h1>

			<p>Welcome to your new appkit server.</p>

			<p>
			  Find instructions on how to set up your app at <a href="http://github.com/theduke/go-appkit">Github</a>
			</p>
		</body>
	</html>
	`

	return []byte(tpl), nil
}

func notFoundHandler(app kit.App, r kit.Request) (kit.Response, bool) {
	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	apiPrefix := "/" + app.Config().UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	// Try to render the page on the server, if enabled.
	if !isApiRequest {
		renderEnabled := app.Config().UBool("serverRenderer.enabled", false)
		noRender := strings.Contains(httpRequest.URL.String(), "no-server-render")

		if renderEnabled && !noRender {
			return serverRenderer(app, r), false
		}
	}

	// For non-api requests, render the default template.
	if !isApiRequest {
		tpl, err := getIndexTpl(app)
		if err != nil {
			return &kit.AppResponse{
				Error: err,
			}, false
		}
		return &kit.AppResponse{
			RawData: tpl,
		}, false
	}

	// For api requests, render the api not found error.
	return &kit.AppResponse{
		Error: kit.AppError{
			Code:    "not_found",
			Message: "This api route does not exist",
		},
	}, false
}

func RespondWithContent(app kit.App, w http.ResponseWriter, code int, content []byte) {
	w.WriteHeader(code)
	w.Write(content)
}

func RespondWithReader(app kit.App, w http.ResponseWriter, code int, reader io.ReadCloser) {
	w.WriteHeader(code)
	io.Copy(w, reader)
	reader.Close()
}

func RespondWithJson(w http.ResponseWriter, response kit.Response) {
	code := 200
	respData := map[string]interface{}{
		"data": response.GetData(),
	}

	if response.GetError() != nil {
		errs := []error{response.GetError()}

		additionalErrs := response.GetError().GetErrors()
		if additionalErrs != nil {
			for _, err := range additionalErrs {
				if apiErr, ok := err.(kit.Error); ok && !apiErr.IsInternal() {
					errs = append(errs, apiErr)
				}
			}
		}

		respData["errors"] = errs
		code = 500
	}

	output, err2 := json.Marshal(respData)
	if err2 != nil {
		code = 500
		respData = map[string]interface{}{
			"errors": []error{
				&kit.AppError{
					Code:    "json_encode_error",
					Message: err2.Error(),
				},
			},
		}
		output, _ = json.Marshal(respData)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(output)
}

func httpHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params, app kit.App, handler kit.RequestHandler) {
	request := kit.NewRequest()

	if r.Body != nil {
		request.BuildFromJsonBody(r)
	}

	request.Context.Set("httpRequest", r)
	request.Context.Set("responseWriter", w)

	for _, param := range params {
		request.Context.Set(param.Key, param.Value)
	}

	var response kit.Response

	// Process all middlewares.
	for _, middleware := range app.BeforeMiddlewares() {
		var skip bool
		response, skip = middleware(app, request)
		if skip {
			return
		} else if response != nil {
			break
		}
	}

	// Only run the handler if no middleware provided a response.
	if response == nil {
		skip := false
		response, skip = handler(app, request)
		if skip {
			return
		}
	}

	// Handle options requests.
	if r.Method == "OPTIONS" {
		header := w.Header()

		allowedOrigins := app.Config().UString("accessControl.allowedOrigins", "*")
		header.Set("Access-Control-Allow-Origin", allowedOrigins)

		methods := app.Config().UString("accessControl.allowedMethods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		header.Set("Access-Control-Allow-Methods", methods)

		allowedHeaders := app.Config().UString("accessControl.allowedHeaders", "Authentication, Content-Type")
		header.Set("Access-Control-Allow-Headers", allowedHeaders)

		response = &kit.AppResponse{
			RawData: []byte{},
		}
	}

	// Note: error responses are converted with the serverErrrorMiddleware middleware.

	for _, middleware := range app.AfterMiddlewares() {
		skip := middleware(app, request, response)
		if skip {
			return
		}
	}

	if response.GetHttpStatus() == 0 {
		response.SetHttpStatus(200)
	}

	// If a data reader is set, write the data of the reader.
	reader := response.GetRawDataReader()
	if reader != nil {
		status := response.GetHttpStatus()
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)
		io.Copy(w, reader)
		reader.Close()
		return
	}

	// If raw data is set, write the raw data.
	rawData := response.GetRawData()
	if rawData != nil {
		status := response.GetHttpStatus()
		if status == 0 {
			status = 200
		}
		w.WriteHeader(status)
		w.Write(rawData)
		return
	}

	// Send json response.
	RespondWithJson(w, response)
}

/**
 * Request trace middlewares.
 */

func RequestTraceMiddleware(a kit.App, r kit.Request) (kit.Response, bool) {
	r.GetContext().Set("startTime", time.Now())
	return nil, false
}

func RequestTraceAfterMiddleware(app kit.App, r kit.Request, response kit.Response) bool {
	r.GetContext().Set("endTime", time.Now())
	return false
}

/**
 * Before middlewares.
 */

func AuthenticationMiddleware(a kit.App, r kit.Request) (kit.Response, bool) {
	// Handle authentication.
	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	if token := httpRequest.Header.Get("Authentication"); token != "" {
		if a.UserService() != nil {
			user, session, err := a.UserService().VerifySession(token)
			if err == nil {
				r.SetUser(user)
				r.SetSession(session)
			} else {
				return &kit.AppResponse{
					Error: err,
				}, false
			}
		}
	}

	return nil, false
}

/**
 * After middlewares.
 */

func ServerErrorMiddleware(app kit.App, r kit.Request, response kit.Response) bool {
	if response.GetError() == nil {
		return false
	}

	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	apiPrefix := "/" + app.Config().UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	response.SetHttpStatus(500)

	data := map[string]interface{}{"errors": []error{response.GetError()}}

	if isApiRequest {
		response.SetData(data)
		return false
	}

	tpl := defaultErrorTpl()

	tplPath := app.Config().UString("frontend.errorTemplate")
	if tplPath != "" {
		t, err := template.ParseFiles(tplPath)
		if err != nil {
			app.Logger().Fatalf("Could not parse error template at '%v': %v", tplPath, err)
		} else {
			tpl = t
		}
	}

	var buffer *bytes.Buffer
	if err := tpl.Execute(buffer, data); err != nil {
		app.Logger().Fatalf("Could not render error template: %v\n", err)
		response.SetRawData([]byte("Server error"))
	} else {
		response.SetRawData(buffer.Bytes())
	}

	return false
}

func RequestLoggerMiddleware(app kit.App, r kit.Request, response kit.Response) bool {
	rawStarted, ok1 := r.GetContext().Get("startTime")
	rawFinished, ok2 := r.GetContext().Get("endTime")

	var timeTaken int64 = int64(-1)
	if ok1 && ok2 {
		started := rawStarted.(time.Time)
		finished := rawFinished.(time.Time)
		timeTaken = int64(finished.Sub(started) / time.Millisecond)
	}

	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	method := httpRequest.Method
	url := httpRequest.URL

	// Log the request.
	if response.GetError() != nil {
		app.Logger().WithFields(log.Fields{
			"action":       "request",
			"method":       method,
			"url":          url.String(),
			"status":       response.GetHttpStatus(),
			"err":          response.GetError(),
			"milliseconds": timeTaken,
		}).Errorf("%v: %v - %v - %v", response.GetHttpStatus(), method, url, response.GetError())
	} else {
		app.Logger().WithFields(log.Fields{
			"action":       "request",
			"method":       method,
			"url":          url.String(),
			"status":       response.GetHttpStatus(),
			"milliseconds": timeTaken,
		}).Debugf("%v: %v - %v", response.GetHttpStatus(), method, url)
	}

	return false
}
