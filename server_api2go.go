package appkit

import (
	"log"
	"errors"
	"net/http"
	"io/ioutil"
	"fmt"
	"encoding/json"

	"github.com/manyminds/api2go"

	db "github.com/theduke/dukedb"
)

func JsonHandler(r *http.Request, app *App, method *Method) (interface{}, ApiError) {
	// Read request body.
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, Error{
			Code: "read_post_error",
			Message: fmt.Sprintf("Request body could not be read: %v", err),
		}
	}

	// Find data and meta.
	rawData := make(map[string]interface{})

	if string(body) == "" {
		rawData["data"] = nil
	} else {
		err = json.Unmarshal(body, &rawData)
		if err != nil {
			return nil, Error{
				Code: "invalid_json_body",
				Message: fmt.Sprintf("POST body json could not be unmarshaled: %v", err),
			}
		}
	}

	// Build the request.
	data, _ := rawData["data"]
	meta := make(map[string]interface{})
	request := buildRequest(app.GetUserHandler(), r.Header, data, meta)

	// Check permissions.
	if method.RequiresUser && request.User == nil {
		return nil, Error{Code: "permission_denied"}
	}

	// Call the method callback.
	responseData, err2 := method.Run(app, request)
	if err != nil {
		return nil, err2
	}

	return responseData, nil
}

func JsonWrapHandler(w http.ResponseWriter, r *http.Request, app *App, method *Method) {
	header := w.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	header.Set("Access-Control-Allow-Headers", "Authentication, Content-Type")

	// Handle options requests.
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		w.Write([]byte(""))
		return
	}

	data, err := JsonHandler(r, app, method)
	code := 200
	
	resp := map[string]interface{}{
		"data": data,
	}	
	
	if err != nil {
		resp["errors"] = []error{err}
		code = 500
	}

	output, err2 := json.Marshal(resp)
	if err2 != nil {
		log.Printf("JSON encode error: %v\n", err)
		code = 500
		resp["errors"] = []error{
			&Error{
				Code: "json_encode_error",
				Message: err.Error(),
			},
		}

		output, err2 = json.Marshal(resp)
		if err2 != nil {
			output = []byte("{errors:[{\"code\": \"json_encode_error\"}]}")
		}
	}

	header.Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(output)
}

func buildRequest(
	userHandler ApiUserHandler, 
	header http.Header,
	data interface{},
	meta map[string]interface{}) *Request {
	
	req := NewRequest()
			
	// Handle authentication.
	if token := header.Get("Authentication"); token != "" {
		if userHandler != nil {
			user, session, err := userHandler.VerifySession(token)	

			if err == nil {
				req.User = user
				req.Session = session
			}
		}
	}
	
	req.Meta = NewContext()
	req.Meta.Data = meta

	req.Data = data

	return req
}

func BuildQuery(r api2go.Request, q *db.Query) (*db.Query, error) {
	return q, nil
}


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

type Api2GoResource struct {
	AppResource ApiResource
	App *App
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



func (res Api2GoResource) FindOne(rawId string, r api2go.Request) (api2go.Responder, error) {
	request := buildRequest(res.App.GetUserHandler(), r.Header, r.Data, r.Meta)
 	response := res.AppResource.ApiFindOne(rawId, request)
 	return convertResult(response, 200)
}

func (res Api2GoResource) FindAll(r api2go.Request) (api2go.Responder, error) {
	q, err := BuildQuery(r, res.AppResource.GetQuery())
	if err != nil {
		return nil, api2go.NewHTTPError(errors.New("invalid_query"), err.Error(), 500)
	}

	request := buildRequest(res.App.GetUserHandler(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiFind(q, request)
 	return convertResult(response, 200)
}

func (res Api2GoResource) Create(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(db.Model)
	request := buildRequest(res.App.GetUserHandler(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiCreate(model, request)
 	return convertResult(response, 201)
}

func (res Api2GoResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(db.Model)
	request := buildRequest(res.App.GetUserHandler(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiUpdate(model, request)
 	return convertResult(response, 200)
}

func (res Api2GoResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
	request := buildRequest(res.App.GetUserHandler(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiDelete(rawId, request)
 	return convertResult(response, 200)
}
