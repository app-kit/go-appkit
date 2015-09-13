package appkit

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"encoding/json"

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

func (r *Request) GetContext() Context {
	return r.Context
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

type HttpRoute struct {
	Route string
	Method string
	Handler HttpHandler
}

func RespondWithJson(w http.ResponseWriter, response ApiResponse) {
	code := 200
	respData := map[string]interface{} {
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
		respData = map[string]interface{} {
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

func httpRequestMiddleware(w http.ResponseWriter, r *http.Request, params httprouter.Params, app *App, handler HttpHandler) {
	request := NewRequest()
	request.HttpRequest = r

	for _, param := range params {
		request.Context.Set(param.Key, param.Value)
	}

	var response ApiResponse

	// Process all middlewares.
	for _, middleware := range app.GetMiddlewares() {
		response, skip := middleware(app, request, w)
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

	RespondWithJson(w, response)	
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
