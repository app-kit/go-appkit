package jsonapi

import (
	"fmt"
	"math"
	"strings"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/utils"
)

func HandleOptions(a kit.App, r kit.Request) (kit.Response, bool) {
	return &kit.AppResponse{}, false
}

func HandleWrap(collection string, handler kit.RequestHandler) kit.RequestHandler {
	return func(a kit.App, r kit.Request) (kit.Response, bool) {
		r.GetContext().Set("collection", collection)
		return handler(a, r)
	}
}

func Find(res kit.Resource, request kit.Request) (kit.Response, apperror.Error) {
	collection := res.Collection()

	info := res.Backend().ModelInfo(collection)

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
		var limit, offset int

		context := request.GetContext()

		if context.Has("limit") {
			val, err := context.Int("limit")
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_limit_parameter",
					Message: "The get query contains a non-numeric ?limit",
				}
			}
			limit = val
		}

		if context.Has("offset") {
			val, err := context.Int("offset")
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_offset_parameter",
					Message: "The get query contains a non-numeric ?offset",
				}
			}
			offset = val
		}

		var page, perPage int

		if context.Has("page") {
			val, err := context.Int("page")
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_page_parameter",
					Message: "The get query contains a non-numeric ?page",
				}
			}
			page = val
		}

		if context.Has("per_page") {
			val, err := context.Int("per_page")
			if err != nil {
				return nil, &apperror.Err{
					Code:    "non_numeric_per_page_parameter",
					Message: "The get query contains a non-numeric ?per_page",
				}
			}
			perPage = val
		}

		if perPage > 0 {
			limit = perPage
		}

		if page > 1 {
			offset = (page - 1) * limit
		}

		if limit > 0 {
			query.Limit(int(limit)).Offset(int(offset))
		}

		// Add joins.
		if context.Has("joins") {
			parts := strings.Split(context.String("joins"), ",")
			for _, name := range parts {
				fieldName := info.MapMarshalName(name)
				if fieldName == "" {
					fieldName = name
				}

				if !info.HasField(fieldName) {
					return nil, &apperror.Err{
						Code:    "invalid_join",
						Message: fmt.Sprintf("Tried to join a NON-existant relationship %v", name),
						Public:  true,
					}
				}

				query.Join(fieldName)
			}
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

	// If response contains a count and the request a "perPage" param, add a total_pages param
	// to meta.
	perPage, err2 := request.GetContext().Int("per_page")

	meta := response.GetMeta()
	if meta != nil && err2 == nil {
		count, ok := meta["count"]
		if ok {
			meta["total_pages"] = math.Ceil(float64(count.(int)) / float64(perPage))
		}
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

	_, fullUpdate := request.GetContext().Get("full-update")
	var response kit.Response

	if fullUpdate {
		response = res.ApiUpdate(model, request)
	} else {
		response = res.ApiPartialUpdate(model, request)
	}

	return ConvertResponse(res.Backend(), response), nil
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
