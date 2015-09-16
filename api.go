package appkit

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"html/template"
	"os"
	"runtime"
	"path"
	"os/exec"
	"regexp"
	"strconv"
	"time"
	"bytes"

	log "github.com/Sirupsen/logrus"
	"github.com/twinj/uuid"
	"github.com/julienschmidt/httprouter"

	. "github.com/theduke/go-appkit/error"
	"github.com/theduke/go-appkit/utils"
	"github.com/theduke/go-appkit/caches"
)

type Context struct {
	Data map[string]interface{}
}

func NewContext() Context {
	c := Context{}
	c.Data = make(map[string]interface{})

	return c
}

func (c Context) Get(key string) (interface{}, bool) {
	x, ok := c.Data[key]
	return x, ok
}

func (c Context) MustGet(key string) interface{} {
	x, ok := c.Data[key]
	if !ok {
		panic("Context does not have key " + key)
	}

	return x
}

func (c *Context) Set(key string, data interface{}) {
	c.Data[key] = data
}

func (c Context) String(key string) string {
	x, ok := c.Data[key]
	if !ok {
		return ""
	}

	str, ok := x.(string)
	if !ok {
		return ""
	}
	return str
}

func (c *Context) SetString(key, val string) {
	c.Data[key] = val
}

type Request struct {
	User    ApiUser
	Session ApiSession

	Context Context
	Meta    Context
	Data    interface{}
}

func NewRequest() *Request {
	r := Request{}
	r.Context = NewContext()
	r.Meta = NewContext()

	return &r
}

func (r *Request) BuildFromJsonBody(request *http.Request) Error {
	// Read request body.
	defer request.Body.Close()
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return AppError{
			Code:    "read_post_error",
			Message: fmt.Sprintf("Request body could not be read: %v", err),
		}
	}

	// Find data and meta.
	allData := make(map[string]interface{})

	if string(body) != "" {
		err = json.Unmarshal(body, &allData)
		if err != nil {
			return AppError{
				Code:    "invalid_json_body",
				Message: fmt.Sprintf("Json body could not be unmarshaled: %v", err),
			}
		}
	}

	if rawData, ok := allData["data"]; ok {
		r.Data = rawData
	}

	if rawMeta, ok := allData["meta"]; ok {
		if meta, ok := rawMeta.(map[string]interface{}); ok {
			r.Meta.Data = meta
		}
	}

	return nil
}

func (r *Request) GetUser() ApiUser {
	return r.User
}

func (r *Request) SetUser(x ApiUser) {
	r.User = x
}

func (r *Request) GetSession() ApiSession {
	return r.Session
}

func (r *Request) SetSession(x ApiSession) {
	r.Session = x
}

func (r *Request) GetContext() *Context {
	return &r.Context
}

func (r *Request) SetContext(x Context) {
	r.Context = x
}

func (r *Request) GetMeta() Context {
	return r.Meta
}

func (r *Request) SetMeta(x Context) {
	r.Meta = x
}

func (r *Request) GetData() interface{} {
	return r.Data
}

func (r *Request) SetData(x interface{}) {
	r.Data = x
}

type Response struct {
	Error Error
	HttpStatus int

	Meta  map[string]interface{}

	Data  interface{}
	RawData []byte
	RawDataReader io.ReadCloser
}

func (r *Response) GetError() Error {
	return r.Error
}

func (r *Response) GetHttpStatus() int {
	return r.HttpStatus
}

func (r *Response) SetHttpStatus(status int) {
	r.HttpStatus = status
}

func (r *Response) GetMeta() map[string]interface{} {
	return r.Meta
}

func (r *Response) SetMeta(m map[string]interface{}) {
	r.Meta = m
}

func (r *Response) GetData() interface{} {
	return r.Data
}

func (r *Response) SetData(data interface{}) {
	r.Data = data
}

func (r *Response)  GetRawData() []byte {
	return r.RawData
}

func (r *Response) SetRawData(data []byte) {
	r.RawData = data
}

func (r *Response) GetRawDataReader() io.ReadCloser {
	return r.RawDataReader
}

func (r *Response) SetRawDataReader(reader io.ReadCloser) {
	r.RawDataReader = reader
}

func NewErrorResponse(code, message string) *Response {
	return &Response{
		Error: AppError{Code: code, Message: message},
	}
}

type RequestHandler func(*App, ApiRequest) (ApiResponse, bool)
type AfterRequestMiddleware func(*App, ApiRequest, ApiResponse) bool

type HttpHandlerStruct struct {
	App *App
	Handler RequestHandler
}

func (h *HttpHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := new(httprouter.Params)
	httpHandler(w, r, *params, h.App, h.Handler)
}

type HttpRoute struct {
	Route   string
	Method  string
	Handler RequestHandler
}

func serverRenderer(app *App, r ApiRequest) ApiResponse {
	url := r.GetContext().MustGet("httpRequest").(*http.Request).URL

	// Build the url to query.
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	if url.Host == "" {
		url.Host = app.Config.UString("host", "localhost") + ":" + app.Config.UString("port", "8000")
	}

	q := url.Query()
	q.Set("no-server-render", "1")
	url.RawQuery = q.Encode()

	strUrl := url.String()

	cacheKey := "serverrenderer_" + strUrl
	cacheName := app.Config.UString("serverRenderer.cache")
	var cache caches.Cache

	// If a cache is specified, try to retrieve it.
	if cacheName != "" {
		cache = app.Cache(cacheName)
		if cache == nil {
			app.Logger.Errorf("serverRenderer.cache is set to %v, but the cache is not registered with app", cacheName)
		}
	}

	// If a cache was found, try to retrieve cached response.
	if cache != nil {
		item, err := cache.Get(cacheKey)
		if err != nil {
			app.Logger.Errorf("serverRenderer: cache retrieval error: %v", err)
		} else if item != nil {
			// Cache item found, return response with cache item.
			status, _ := strconv.ParseInt(item.GetTags()[0], 10, 64)
			data, _ := item.ToString()

			return &Response{
				HttpStatus: int(status),
				RawData: []byte(data),
			}
		}
	}

	// Either no cache or url not yet cached, so render it.

	// First, ensure that the tmp directory exists.
	tmpDir := path.Join(app.TmpDir(), "phantom")
	if ok, _ := utils.FileExists(tmpDir); !ok {
		if err := os.MkdirAll(tmpDir, 0777); err != nil {
			return &Response{
				Error: AppError{
					Code: "create_tmp_dir_failed",
					Message: fmt.Sprintf("Could not create the tmp directory at %v: %v", tmpDir, err),
					Internal: true,
				},
			}
		}
	}

	// Build a unique file name.
	filePath := path.Join(tmpDir, uuid.NewV4().String() + ".html")

	// Execute phantom js.

	// Find path of phantom script.
	_, filename, _, _ := runtime.Caller(1)
	scriptPath := path.Join(path.Dir(filename), "phantom", "render.js")

	start := time.Now()

	phantomPath := app.Config.UString("serverRenderer.phantomJsPath", "phantomjs")
	
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
		app.Logger.Errorf("Phantomjs execution error: %v", string(result))

		return &Response{
			Error: AppError{
				Code: "phantom_execution_failed",
				Message: err.Error(),
				Data: result,
				Errors: []error{err},
				Internal: true,
			},
		}
	}

	// Get time taken as milliseconds.
	timeTaken := int(time.Now().Sub(start) / time.Millisecond)
	app.Logger.WithFields(log.Fields{
		"action": "phantomjs_render",
		"milliseconds": timeTaken,
	}).Debugf("Rendered url %v with phantomjs", url)

	content, err2 := utils.ReadFile(filePath)
	if err2 != nil {
		return &Response{Error: err2}
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
		lifetime := app.Config.UInt("serverRenderer.cacheLiftetime", 3600)

		err := cache.Set(&caches.StrItem{
			Key: cacheKey,
			Value: string(content),
			Tags: []string{strconv.FormatInt(int64(status), 10)},
			ExpiresAt: time.Now().Add(time.Duration(lifetime) * time.Second),
		})
		if err != nil {
			app.Logger.Errorf("serverRenderer: Cache persist error: %v",  err)
		}
	}

	return &Response{
		HttpStatus: status,
		RawData: content,
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

func getIndexTpl(app *App) ([]byte, Error) {
	if path := app.Config.UString("frontend.indexTpl"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, AppError{
				Code: "cant_open_index_tpl",
				Message: fmt.Sprintf("The index template at %v could not be opened: %v", path, err),
			}
		}

		tpl, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, AppError{
				Code: "index_tpl_read_error",
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

func notFoundHandler(app *App, r ApiRequest) (ApiResponse, bool) {
	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	apiPrefix := "/" + app.Config.UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	// Try to render the page on the server, if enabled.
	if !isApiRequest {
		renderEnabled := app.Config.UBool("serverRenderer.enabled", false)
		noRender := strings.Contains(httpRequest.URL.String(), "no-server-render")

		if renderEnabled && !noRender {
			return serverRenderer(app, r), false
		}
	}

	// For non-api requests, render the default template.
	if !isApiRequest {
		tpl, err := getIndexTpl(app)
		if err != nil {
			return &Response{
				Error: err,
			}, false
		}
		return &Response{
			RawData: tpl,
		}, false
	}

	// For api requests, render the api not found error.
	return &Response{
		Error: AppError{
			Code: "not_found",
			Message: "This api route does not exist",
		},
	}, false
}

func RespondWithContent(app *App, w http.ResponseWriter, code int, content []byte) {
	w.WriteHeader(code)
	w.Write(content)
}

func RespondWithReader(app *App, w http.ResponseWriter, code int, reader io.ReadCloser) {
	w.WriteHeader(code)
	io.Copy(w, reader)
	reader.Close()
}

func RespondWithJson(w http.ResponseWriter, response ApiResponse) {
	code := 200
	respData := map[string]interface{}{
		"data": response.GetData(),
	}

	if response.GetError() != nil {
		errs := []error{response.GetError()}

		additionalErrs := response.GetError().GetErrors()
		if additionalErrs != nil {
			for _, err := range additionalErrs {
				if apiErr, ok := err.(Error); ok && !apiErr.IsInternal() {
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
				&AppError{
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

func httpHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params, app *App, handler RequestHandler) {
	request := NewRequest()

	request.Context.Set("httpRequest", r)
	request.Context.Set("responseWriter", w)

	for _, param := range params {
		request.Context.Set(param.Key, param.Value)
	}

	var response ApiResponse

	// Process all middlewares.
	for _, middleware := range app.GetBeforeMiddlewares() {
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

		allowedOrigins := app.Config.UString("accessControl.allowedOrigins", "*")
		header.Set("Access-Control-Allow-Origin", allowedOrigins)

		methods := app.Config.UString("accessControl.allowedMethods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		header.Set("Access-Control-Allow-Methods", methods)

		allowedHeaders := app.Config.UString("accessControl.allowedHeaders", "Authentication, Content-Type")
		header.Set("Access-Control-Allow-Headers", allowedHeaders)

		response = &Response{
			RawData: []byte{},
		}
	}

	// Note: error responses are converted with the serverErrrorMiddleware middleware.

	for _, middleware := range app.GetAfterMiddlewares() {
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

func RequestTraceMiddleware(a *App, r ApiRequest) (ApiResponse, bool) {
	r.GetContext().Set("startTime", time.Now())
	return nil, false
}

func RequestTraceAfterMiddleware(app *App, r ApiRequest, response ApiResponse) bool {
	r.GetContext().Set("endTime", time.Now())
	return false
}

/**
 * Before middlewares.
 */

func AuthenticationMiddleware(a *App, r ApiRequest) (ApiResponse, bool) {
	// Handle authentication.
	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	if token := httpRequest.Header.Get("Authentication"); token != "" {
		if a.GetUserHandler() != nil {
			user, session, err := a.GetUserHandler().VerifySession(token)
			if err == nil {
				r.SetUser(user)
				r.SetSession(session)
			} else {
				return &Response{
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

func ServerErrorMiddleware(app *App, r ApiRequest, response ApiResponse) bool {
	if response.GetError() == nil {
		return false
	}

	httpRequest := r.GetContext().MustGet("httpRequest").(*http.Request)
	apiPrefix := "/" + app.Config.UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(httpRequest.URL.Path, apiPrefix)

	response.SetHttpStatus(500)	

	data := map[string]interface{}{"errors": []error{response.GetError()}}

	if isApiRequest {
		response.SetData(data)
		return false
	}

	tpl := defaultErrorTpl()

	tplPath := app.Config.UString("frontend.errorTemplate")
	if tplPath != "" {
		t, err := template.ParseFiles(tplPath)
		if err != nil {
			app.Logger.Fatalf("Could not parse error template at '%v': %v", tplPath, err)
		} else {
			tpl = t
		}
	}

	var buffer *bytes.Buffer
	if err := tpl.Execute(buffer, data); err != nil {
		app.Logger.Fatalf("Could not render error template: %v\n", err)
		response.SetRawData([]byte("Server error"))
	} else {
		response.SetRawData(buffer.Bytes())
	}

	return false
}

func RequestLoggerMiddleware(app *App, r ApiRequest, response ApiResponse) bool {
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
		app.Logger.WithFields(log.Fields{
			"action": "request",
			"method": method,
			"url": url.String(),
			"status": response.GetHttpStatus(),
			"err": response.GetError(),
			"milliseconds": timeTaken,
		}).Errorf("%v: %v - %v - %v", response.GetHttpStatus(), method, url, response.GetError())
	} else {
		app.Logger.WithFields(log.Fields{
			"action": "request",
			"method": method,
			"url": url.String(),
			"status": response.GetHttpStatus(),
			"milliseconds": timeTaken,
		}).Debugf("%v: %v - %v", response.GetHttpStatus(), method, url)
	}

	return false
}
