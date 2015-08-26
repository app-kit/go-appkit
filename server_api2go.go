package appkit

import (
	"log"
	"errors"

	"github.com/manyminds/api2go"

	db "github.com/theduke/dukedb"
)

type Api2GoResponse struct {
	Res  interface{}
	Status int
	Meta map[string]interface{}
}

// Metadata returns additional meta data
func (r Api2GoResponse) Metadata() map[string]interface{} {
	return r.Meta
}

// Result returns the actual payload
func (r Api2GoResponse) Result() interface{} {
	return r.Res
}

// StatusCode sets the return status code
func (r Api2GoResponse) StatusCode() int {
	return r.Status
}

func BuildQuery(r api2go.Request, q *db.Query) (*db.Query, error) {
	return q, nil
}

type Api2GoResource struct {
	AppResource ApiResource
}

func convertResult(res ApiResponse, status int) (api2go.Responder, error) {
	if err := res.GetError(); err != nil {
		status := 500
		if err.GetCode() == "not_found" || err.GetCode() == "record_not_found" {
			status = 404
		}
		return nil, api2go.NewHTTPError(errors.New(err.GetCode()), err.GetMessage(), status)
	}
	return &Api2GoResponse{
		Res: res.GetData(),
		Meta: res.GetMeta(),
		Status: status,
	}, nil
}

func (res Api2GoResource) buildRequest(r api2go.Request) *Request {
	req := NewRequest()
			
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
	
	req.Meta = NewContext()
	req.Meta.Data = r.Meta

	return req
}

func (res Api2GoResource) FindOne(rawId string, r api2go.Request) (api2go.Responder, error) {
 	response := res.AppResource.ApiFindOne(rawId, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) FindAll(r api2go.Request) (api2go.Responder, error) {
	q, err := BuildQuery(r, res.AppResource.GetQuery())
	if err != nil {
		return nil, api2go.NewHTTPError(errors.New("invalid_query"), err.Error(), 500)
	}

	response := res.AppResource.ApiFind(q, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) Create(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	log.Printf("creating user %v\n", obj)
	model := obj.(db.Model)
	response := res.AppResource.ApiCreate(model, res.buildRequest(r))
 	return convertResult(response, 201)
}

func (res Api2GoResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(db.Model)
	response := res.AppResource.ApiUpdate(model, res.buildRequest(r))
 	return convertResult(response, 200)
}

func (res Api2GoResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
	response := res.AppResource.ApiDelete(rawId, res.buildRequest(r))
 	return convertResult(response, 200)
}

