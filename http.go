package appkit

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/theduke/go-apperror"
)

type AppRequest struct {
	Frontend   string
	Path       string
	HttpMethod string

	Context *Context
	Meta    *Context

	RawData []byte
	Data    interface{}

	User    User
	Session Session

	HttpRequest        *http.Request
	HttpResponseWriter http.ResponseWriter
}

func NewRequest() *AppRequest {
	r := AppRequest{}
	r.Context = NewContext()
	r.Meta = NewContext()

	return &r
}

func (r *AppRequest) GetFrontend() string {
	return r.Frontend
}

func (r *AppRequest) SetFrontend(x string) {
	r.Frontend = x
}

func (r *AppRequest) GetPath() string {
	return r.Path
}

func (r *AppRequest) SetPath(x string) {
	r.Path = x
}

func (r *AppRequest) GetHttpMethod() string {
	return r.HttpMethod
}

func (r *AppRequest) SetHttpMethod(x string) {
	r.HttpMethod = x
}

func (r *AppRequest) GetUser() User {
	return r.User
}

func (r *AppRequest) SetUser(x User) {
	r.User = x
}

func (r *AppRequest) GetSession() Session {
	return r.Session
}

func (r *AppRequest) SetSession(x Session) {
	r.Session = x
}

func (r *AppRequest) GetContext() *Context {
	return r.Context
}

func (r *AppRequest) SetContext(x *Context) {
	r.Context = x
}

func (r *AppRequest) GetMeta() *Context {
	return r.Meta
}

func (r *AppRequest) SetMeta(x *Context) {
	r.Meta = x
}

func (r *AppRequest) GetData() interface{} {
	return r.Data
}

func (r *AppRequest) SetData(x interface{}) {
	r.Data = x
}

func (r *AppRequest) GetRawData() []byte {
	return r.RawData
}

func (r *AppRequest) SetRawData(data []byte) {
	r.RawData = data
}

func (r *AppRequest) GetHttpRequest() *http.Request {
	return r.HttpRequest
}

func (r *AppRequest) SetHttpRequest(request *http.Request) {
	r.HttpRequest = request
}

func (r *AppRequest) GetHttpResponseWriter() http.ResponseWriter {
	return r.HttpResponseWriter
}

func (r *AppRequest) SetHttpResponseWriter(writer http.ResponseWriter) {
	r.HttpResponseWriter = writer
}

func (r *AppRequest) ReadHttpBody() apperror.Error {
	if r.HttpRequest == nil {
		return apperror.New("no_http_request")
	}

	if r.HttpRequest.Body == nil {
		return nil
	}

	// Read request body.
	defer r.HttpRequest.Body.Close()
	body, err := ioutil.ReadAll(r.HttpRequest.Body)
	if err != nil {
		return apperror.Wrap(err, "http_body_read_error", "Could not read http body")
	}

	r.RawData = body
	return nil
}

func (r *AppRequest) ParseJsonData() apperror.Error {
	if r.RawData == nil {
		return apperror.New("no_raw_data")
	}

	if string(r.RawData) == "" {
		// Skip with empty body.
		return nil
	}

	// Find data and meta.
	allData := make(map[string]interface{})

	if err := json.Unmarshal(r.RawData, &allData); err != nil {
		return apperror.Wrap(err, "invalid_json_body", "JSON in body could not be unmarshalled")
	}

	r.Data = allData

	return nil
}

// Unserialize converts the raw request data with the given serializer.
func (r *AppRequest) Unserialize(serializer Serializer) apperror.Error {
	return serializer.UnserializeRequest(r.Data, r)
}

type AppResponse struct {
	Error      apperror.Error
	HttpStatus int

	Meta map[string]interface{}

	Data          interface{}
	RawData       []byte
	RawDataReader io.ReadCloser
}

func (r *AppResponse) GetError() apperror.Error {
	return r.Error
}

func (r *AppResponse) GetHttpStatus() int {
	return r.HttpStatus
}

func (r *AppResponse) SetHttpStatus(status int) {
	r.HttpStatus = status
}

func (r *AppResponse) GetMeta() map[string]interface{} {
	return r.Meta
}

func (r *AppResponse) SetMeta(m map[string]interface{}) {
	r.Meta = m
}

func (r *AppResponse) GetData() interface{} {
	return r.Data
}

func (r *AppResponse) SetData(data interface{}) {
	r.Data = data
}

func (r *AppResponse) GetRawData() []byte {
	return r.RawData
}

func (r *AppResponse) SetRawData(data []byte) {
	r.RawData = data
}

func (r *AppResponse) GetRawDataReader() io.ReadCloser {
	return r.RawDataReader
}

func (r *AppResponse) SetRawDataReader(reader io.ReadCloser) {
	r.RawDataReader = reader
}

// All arguments are passed to apperror.New(). Check apperror docs for more info.
func NewErrorResponse(args ...interface{}) *AppResponse {
	if len(args) == 0 {
		panic("Must supply at least an apperror.Error or a string (error code)")
	}

	firstArg := args[0]
	if err, ok := firstArg.(apperror.Error); ok {
		return &AppResponse{Error: err}
	} else if code, ok := firstArg.(string); ok {
		return &AppResponse{
			Error: apperror.New(code, args[1:]...),
		}
	} else {
		panic("Invalid first argument: must be apperror.Error or string (error code)")
	}
}

type AppHttpRoute struct {
	route   string
	method  string
	handler RequestHandler
}

func (r *AppHttpRoute) Route() string {
	return r.route
}

func (r *AppHttpRoute) Method() string {
	return r.method
}

func (r *AppHttpRoute) Handler() RequestHandler {
	return r.handler
}

func NewHttpRoute(route, method string, handler RequestHandler) *AppHttpRoute {
	return &AppHttpRoute{
		route:   route,
		method:  method,
		handler: handler,
	}
}
