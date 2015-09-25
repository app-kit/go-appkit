package jsonapi

import (
	"fmt"
	"strconv"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/utils"
)

func HandleWrap(collection string, handler kit.RequestHandler) kit.RequestHandler {
	return func(a kit.App, r kit.Request) (kit.Response, bool) {
		fmt.Printf("collectioN: %v\n", collection)
		r.GetContext().Set("collection", collection)
		return handler(a, r)
	}
}

func Find(res kit.Resource, request kit.Request) (kit.Response, apperror.Error) {
	collection := res.Collection()
	if err := request.ParseJsonData(); err != nil {
		return nil, err
	}

	var query db.Query

	jsonQuery := utils.GetMapStringKey(request.GetData(), "query")
	if jsonQuery != "" {
		// A custom query was supplied.
		// Try to parse the query.
		var err apperror.Error
		query, err = db.ParseJsonQuery(collection, []byte(jsonQuery))
		if err != nil {
			return nil, apperror.Wrap(err, "invalid_query", "")
		}
	}

	if query == nil {
		query = db.Q(collection)

		// No custom query.
		// Check paging parameters.
		var limit, offset int64
		if rawLimit := request.GetContext().String("limit"); rawLimit != "" {
			var err error
			limit, err = strconv.ParseInt(rawLimit, 10, 64)
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_limit_parameter",
					Message: "The get query contains a non-numeric ?limit",
				}
			}
		}
		if rawOffset := request.GetContext().String("offset"); rawOffset != "" {
			var err error
			offset, err = strconv.ParseInt(rawOffset, 10, 64)
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_offset_parameter",
					Message: "The get query contains a non-numeric ?offset",
				}
			}
		}

		if limit > 0 {
			query.Limit(int(limit)).Offset(int(offset))
		}
	}

	return res.ApiFind(query, request), nil
}

func HandleFind(app kit.App, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")

	res := app.Dependencies().Resource(collection)
	if res == nil || !res.IsPublic() {
		err := &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
		return &kit.AppResponse{Error: err}, false
	}

	response, err := Find(res, request)
	if err != nil {
		response = &kit.AppResponse{Error: err}
	}

	return ConvertResponse(res.Backend(), response), false
}

func HandleFindOne(app kit.App, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")
	id := request.GetContext().MustString("id")

	res := app.Dependencies().Resource(collection)
	if res == nil || !res.IsPublic() {
		resp := kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The resource '%v' does not exist", collection))
		return ConvertResponse(res.Backend(), resp), false
	}

	return ConvertResponse(res.Backend(), res.ApiFindOne(id, request)), false
}

func Create(app kit.App, request kit.Request) (kit.Response, apperror.Error) {
	if err := request.ParseJsonData(); err != nil {
		return nil, err
	}

	collection := request.GetContext().MustString("collection")

	res := app.Dependencies().Resource(collection)
	if res == nil || !res.IsPublic() {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
	}

	model, err := BuildModel(res.Backend(), collection, request.GetRawData())
	if err != nil {
		return nil, err
	}

	response := res.ApiCreate(model, request)
	if response.GetError() == nil {
		response.SetHttpStatus(201)
	}
	response = ConvertResponse(res.Backend(), response)
	return response, nil
}

func HandleCreate(app kit.App, request kit.Request) (kit.Response, bool) {
	response, err := Create(app, request)
	if err != nil {
		return ConvertResponse(nil, &kit.AppResponse{Error: err}), false
	}

	return response, false
}

func Update(app kit.App, request kit.Request) (kit.Response, apperror.Error) {
	if err := request.ParseJsonData(); err != nil {
		return nil, err
	}

	collection := request.GetContext().MustString("collection")

	res := app.Dependencies().Resource(collection)
	if res == nil || !res.IsPublic() {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
	}

	model, err := BuildModel(res.Backend(), "", request.GetRawData())
	if err != nil {
		return nil, err
	}

	response := ConvertResponse(res.Backend(), res.ApiUpdate(model, request))
	return response, nil
}

func HandleUpdate(app kit.App, request kit.Request) (kit.Response, bool) {
	response, err := Update(app, request)
	if err != nil {
		return ConvertResponse(nil, &kit.AppResponse{Error: err}), false
	}

	return response, false
}

func HandleDelete(app kit.App, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")
	id := request.GetContext().MustString("id")

	res := app.Dependencies().Resource(collection)
	if res == nil || !res.IsPublic() {
		resp := kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The resource '%v' does not exist", collection))
		return ConvertResponse(res.Backend(), resp), false
	}

	return ConvertResponse(res.Backend(), res.ApiDelete(id, request)), false
}
