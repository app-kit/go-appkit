package appkit

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	. "github.com/theduke/go-appkit/error"
)

type AppRequest struct {
	Context Context
	Meta    Context
	Data    interface{}

	User    User
	Session Session
}

func NewRequest() *AppRequest {
	r := AppRequest{}
	r.Context = NewContext()
	r.Meta = NewContext()

	return &r
}

func (r *AppRequest) BuildFromJsonBody(request *http.Request) Error {
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
	return &r.Context
}

func (r *AppRequest) SetContext(x Context) {
	r.Context = x
}

func (r *AppRequest) GetMeta() Context {
	return r.Meta
}

func (r *AppRequest) SetMeta(x Context) {
	r.Meta = x
}

func (r *AppRequest) GetData() interface{} {
	return r.Data
}

func (r *AppRequest) SetData(x interface{}) {
	r.Data = x
}

type AppResponse struct {
	Error      Error
	HttpStatus int

	Meta map[string]interface{}

	Data          interface{}
	RawData       []byte
	RawDataReader io.ReadCloser
}

func (r *AppResponse) GetError() Error {
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

func NewErrorResponse(code, message string) *AppResponse {
	return &AppResponse{
		Error: AppError{Code: code, Message: message},
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
