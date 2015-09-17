package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/manyminds/api2go"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/error"
)

func JsonHandler(r *http.Request, app kit.App, method *Method) (interface{}, Error) {
	// Read request body.
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, AppError{
			Code:    "read_post_error",
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
			return nil, AppError{
				Code:    "invalid_json_body",
				Message: fmt.Sprintf("POST body json could not be unmarshaled: %v", err),
			}
		}
	}

	// Build the request.
	data, _ := rawData["data"]
	meta := make(map[string]interface{})
	request := buildRequest(app.UserService(), r.Header, data, meta)

	// Check permissions.
	if method.RequiresUser() && request.GetUser() == nil {
		return nil, AppError{Code: "permission_denied"}
	}

	// Call the method callback.
	/*
		responseData, err2 := method.Run(app, request)
		if err != nil {
			return nil, err2
		}

		return responseData, nil
	*/
	return nil, nil
}

func JsonWrapHandler(w http.ResponseWriter, r *http.Request, app kit.App, method *Method) {
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
			&AppError{
				Code:    "json_encode_error",
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
	userService kit.UserService,
	header http.Header,
	data interface{},
	meta map[string]interface{}) kit.Request {

	req := kit.NewRequest()

	// Handle authentication.
	if token := header.Get("Authentication"); token != "" {
		if userService != nil {
			user, session, err := userService.VerifySession(token)
			if err == nil {
				req.User = user
				req.Session = session
			} else {
				log.Printf("Could not verify session: %v\n", err)
			}
		}
	}

	req.Meta = kit.NewContext()
	req.Meta.Data = meta

	req.Data = data

	return req
}

type Api2GoResponse struct {
	Res    interface{}
	Status int
	Meta   map[string]interface{}
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
	AppResource kit.Resource
	App         kit.App
}

func (res Api2GoResource) buildQuery(r api2go.Request) (*db.Query, Error) {

	// Handle a json query in metadata.
	if r.Meta != nil {
		if rawQuery, ok := r.Meta["query"]; ok {
			queryData, ok := rawQuery.(map[string]interface{})
			if !ok {
				return nil, AppError{
					Code:    "invalid_query",
					Message: "Expected query to be dictionary",
				}
			}

			collection := res.AppResource.Model().Collection()
			query, err := db.ParseQuery(collection, queryData)
			if err != nil {
				return nil, err
			}

			return query, nil
		}
	}

	// Not a custom query.
	q := res.AppResource.Q()

	// Handle paging params.
	rawPerPage := r.QueryParams.Get("page[size]")
	rawPage := r.QueryParams.Get("page[number]")

	if rawPerPage != "" && rawPage != "" {
		var page, perPage int

		perPage, _ = strconv.Atoi(rawPerPage)
		page, _ = strconv.Atoi(rawPage)

		q = q.Limit(perPage)
		if page > 1 {
			q = q.Offset((page - 1) * perPage)
		}
	}

	if sort := r.QueryParams["sort"]; sort != nil && sort[0] != "" {
		parts := strings.Split(sort[0], ",")
		for _, part := range parts {
			order := part
			asc := true

			subParts := strings.Split(part, " ")
			if len(subParts) == 2 {
				order = subParts[0]
				asc = subParts[1] == "asc"
			}

			q = q.Order(order, asc)
		}
	}

	return q, nil
}

func (r Api2GoResource) convertResult(res kit.Response, status int) (api2go.Responder, error) {
	if err := res.GetError(); err != nil {
		status := 500
		if err.GetCode() == "not_found" || err.GetCode() == "record_not_found" {
			status = 404
		}
		if err.GetCode() == "permission_denied" {
			status = 403
		}
		return nil, api2go.NewHTTPError(errors.New(err.GetCode()), err.GetMessage(), status)
	}

	response := &Api2GoResponse{
		Res:    res.GetData(),
		Meta:   res.GetMeta(),
		Status: status,
	}

	data := res.GetData()
	if apiModel, ok := data.(Api2GoModel); ok {
		apiModel.SetFullModel(data.(db.Model))
		info := r.AppResource.Backend().GetModelInfo(r.AppResource.Model().Collection())
		apiModel.SetModelInfo(info)
	}

	return response, nil
}

func (res Api2GoResource) FindOne(rawId string, r api2go.Request) (api2go.Responder, error) {
	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiFindOne(rawId, request)
	return res.convertResult(response, 200)
}

func (res Api2GoResource) FindAll(r api2go.Request) (api2go.Responder, error) {
	q, err := res.buildQuery(r)
	if err != nil {
		return nil, api2go.NewHTTPError(errors.New("invalid_query"), err.Error(), 500)
	}

	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiFind(q, request)
	return res.convertResult(response, 200)
}

func (res Api2GoResource) PaginatedFindAll(r api2go.Request) (uint, api2go.Responder, error) {
	q, err := res.buildQuery(r)
	if err != nil {
		return 0, nil, api2go.NewHTTPError(errors.New("invalid_query"), err.Error(), 500)
	}

	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiFindPaginated(q, request)

	var count uint64
	if response.GetError() == nil {
		count = response.GetMeta()["count"].(uint64)
	}

	apiResp, err2 := res.convertResult(response, 200)

	return uint(count), apiResp, err2
}

func (res Api2GoResource) Create(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(db.Model)
	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiCreate(model, request)
	return res.convertResult(response, 201)
}

func (res Api2GoResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	model := obj.(db.Model)
	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiUpdate(model, request)
	return res.convertResult(response, 200)
}

func (res Api2GoResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
	request := buildRequest(res.App.UserService(), r.Header, r.Data, r.Meta)
	response := res.AppResource.ApiDelete(rawId, request)
	return res.convertResult(response, 200)
}
