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

	"github.com/julienschmidt/httprouter"
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

	HttpRequest *http.Request
}

func NewRequest() *Request {
	r := Request{}
	r.Context = NewContext()
	r.Meta = NewContext()

	return &r
}

func (r *Request) BuildFromJsonBody() ApiError {
	// Read request body.
	defer r.HttpRequest.Body.Close()
	body, err := ioutil.ReadAll(r.HttpRequest.Body)
	if err != nil {
		return Error{
			Code:    "read_post_error",
			Message: fmt.Sprintf("Request body could not be read: %v", err),
		}
	}

	// Find data and meta.
	allData := make(map[string]interface{})

	if string(body) != "" {
		err = json.Unmarshal(body, &allData)
		if err != nil {
			return Error{
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

func (r *Request) GetHttpRequest() *http.Request {
	return r.HttpRequest
}

type Response struct {
	Error ApiError
	Meta  map[string]interface{}
	Data  interface{}
}

func (r Response) GetError() ApiError {
	return r.Error
}

func (r Response) GetMeta() map[string]interface{} {
	return r.Meta
}

func (r *Response) SetMeta(m map[string]interface{}) {
	r.Meta = m
}

func (r Response) GetData() interface{} {
	return r.Data
}

func NewErrorResponse(code, message string) *Response {
	return &Response{
		Error: Error{Code: code, Message: message},
	}
}

type HttpHandler func(*App, ApiRequest, http.ResponseWriter) (ApiResponse, bool)

type HttpHandlerStruct struct {
	App *App
	Handler HttpHandler
}

func (h *HttpHandlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := new(httprouter.Params)
	httpHandler(w, r, *params, h.App, h.Handler)
}

type HttpRoute struct {
	Route   string
	Method  string
	Handler HttpHandler
}

func serverRenderer(app *App, w http.ResponseWriter, r ApiRequest) ApiError {
	return nil
}

func defaultErrorTpl() *template.Template {
	tpl := `
	<html>
		<head>
			<title>Server Error</title>
		</head>

		<body>
			<h1>Server Error</h1>

			<p>{{error}}</p>
		</body>
	</html>
	`

	t, _ := template.New("error").Parse(tpl)	
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

func getIndexTpl(app *App) ([]byte, ApiError) {
	if path := app.Config.UString("frontend.indexTpl"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, Error{
				Code: "cant_open_index_tpl",
				Message: fmt.Sprintf("The index template at %v could not be opened: %v", path, err),
			}
		}

		tpl, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, Error{
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

func serverErrorHandler(app *App, r ApiRequest, w http.ResponseWriter) (ApiResponse, bool) {
	tplPath := app.Config.UString("frontend.errorTemplate")

	tpl := defaultErrorTpl()

	if tplPath != "" {
		var err error
		tpl, err = template.ParseFiles(tplPath)
		if err != nil {
			app.Logger.Fatalf("Could not parse error template at '%v': %v", tplPath, err)
		}
	}

	err, _ := r.GetContext().Get("error")
	data := map[string]interface{} {
		"error": err,
	}

	w.WriteHeader(500)
	if err := tpl.Execute(w, data); err != nil {
		app.Logger.Fatalf("Could not render error template: %v\n", err)
		w.Write([]byte("Server Error"))
	}

	return nil, true
}

func notFoundHandler(app *App, r ApiRequest, w http.ResponseWriter) (ApiResponse, bool) {
	apiPrefix := "/" + app.Config.UString("api.prefix", "api")
	isApiRequest := strings.HasPrefix(r.GetHttpRequest().URL.Path, apiPrefix)

	// Try to render the page on the server, if enabled.
	if !isApiRequest {
		if app.Config.UBool("serverRenderer.enabled", false) {
			err := serverRenderer(app, w, r)
			if err == nil {
				// Rendering worked fine and the serverRenderer sent the response.
				// Nothing more to do.
				return  nil, true
			} else {
				// An error occurred, send an error response.
				context := r.GetContext()
				context.Set("error", err)
				app.ServerErrorHandler()(app, r, w)
				return nil, true
			}
		}
	}

	// For non-api requests, render the default template.
	if !isApiRequest {
		tpl, err := getIndexTpl(app)
		if err != nil {
			context := r.GetContext()
			context.Set("error", err)
			app.ServerErrorHandler()(app, r, w)
			return nil, true
		}

		w.WriteHeader(200)
		w.Write(tpl)
		return nil, true
	}

	// Forapi requests, render the api not found error.
	response := &Response{
		Error: Error{
			Code: "not_found",
			Message: "This api route does not exist",
		},
	}
	RespondWithJson(w, response)
	return nil, true
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
				if apiErr, ok := err.(ApiError); ok && !apiErr.IsInternal() {
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
				&Error{
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

func httpHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params, app *App, handler HttpHandler) {
	request := NewRequest()
	request.HttpRequest = r

	for _, param := range params {
		request.Context.Set(param.Key, param.Value)
	}

	var response ApiResponse

	// Process all middlewares.
	for _, middleware := range app.GetBeforeMiddlewares() {
		var skip bool
		response, skip = middleware(app, request, w)
		if skip {
			return
		} else if response != nil {
			break
		}
	}

	// Only run the handler if no middleware provided a response.
	if response == nil {
		skip := false
		response, skip = handler(app, request, w)
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

		w.WriteHeader(200)
		w.Write([]byte(""))
		return
	}

	for _, middleware := range app.GetAfterMiddlewares() {
		var skip bool
		response, skip = middleware(app, request, w)
		if skip {
			return
		} else if response != nil {
			break
		}
	}

	// If an error has occurred and the request is a non-api request,
	// use the app.ServerErrorHandler to respond.
	// Otherwise, do a json response.
	apiPrefix := "/" + app.Config.UString("api.prefix", "api")
	fmt.Printf("path: %v\n", r.URL.Path)
	if response.GetError() != nil && !strings.HasPrefix(r.URL.Path, apiPrefix) {
		request.Context.Set("error", response.GetError())
		app.ServerErrorHandler()(app, request, w)
	} else {
		RespondWithJson(w, response)
	}
}

func AuthenticationMiddleware(a *App, r ApiRequest, w http.ResponseWriter) (ApiResponse, bool) {
	// Handle authentication.
	if token := r.GetHttpRequest().Header.Get("Authentication"); token != "" {
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
