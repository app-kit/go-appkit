package servers

import (
	"errors"

	"github.com/manyminds/api2go"

	"github.com/theduke/appkit"
)

type Response struct {
	Res  interface{}
	Status int
	Meta map[string]interface{}
}

// Metadata returns additional meta data
func (r Response) Metadata() map[string]interface{} {
	return r.Meta
}

// Result returns the actual payload
func (r Response) Result() interface{} {
	return r.Res
}

// StatusCode sets the return status code
func (r Response) StatusCode() int {
	return r.Status
}


type Api2GoResource struct {
	AppResource appkit.ApiResource
}

func convertResult(res appkit.ApiResponse, status int) (api2go.Responder, error) {
	if err := res.GetError(); err != nil {
		status := 500
		if err.GetCode() == "not_found" || err.GetCode() == "record_not_found" {
			status = 403
		}
		return nil, api2go.NewHTTPError(errors.New(err.GetCode()), err.GetMessage(), status)
	}
	return &Response{
		Res: res.GetData(),
		Meta: res.GetMeta(),
		Status: status,
	}, nil
}

func (res Api2GoResource) buildRequest(r api2go.Request) *appkit.Request {
	req := appkit.NewRequest()
			
	// Handle authentication.
	if token := r.Header.Get("Authentication"); token != "" {
		userHandler := res.AppResource.GetUserHandler()
		if userHandler != nil {
			user, session, err := userHandler.VerifySession(token)	

			if err == nil {
				req.User = user
				req.Session = session
			}
		}
	}

	return req
}

func (res Api2GoResource) FindOne(rawId string, r api2go.Request) (api2go.Responder, error) {
 	response := res.AppResource.ApiFindOne(rawId, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) FindAll(r api2go.Request) (api2go.Responder, error) {
	q := appkit.RawQuery{}
	response := res.AppResource.ApiFind(q, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) Create(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(appkit.ApiModel)
	response := res.AppResource.ApiCreate(model, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
	response := res.AppResource.ApiDelete(rawId, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(appkit.ApiModel)
	response := res.AppResource.ApiUpdate(model, res.buildRequest(r))
 	return convertResult(response, 200)
}
