package appkit

import ()

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

func (c Context) GetString(key string) string {
	x, ok := c.Data[key]
	if !ok {
		return ""
	}
	return x.(string)
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
