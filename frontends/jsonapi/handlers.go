package jsonapi

import (
	"fmt"
	"math"
	"strings"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
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

	jsonQuery := request.GetContext().String("query")
	if jsonQuery != "" {
		// A custom query was supplied.
		// Try to parse the query.
		var err apperror.Error
		query, err = db.ParseJsonQuery(collection, []byte(jsonQuery))
		if err != nil {
			return nil, apperror.Wrap(err, "invalid_query", "", true)
		}
	}

	if query == nil {
		query = db.Q(collection)
	}

	// Check paging parameters.
	var limit, offset int

	context := request.GetContext()

	if context.Has("limit") {
		val, err := context.Int("limit")
		if err != nil {
			return nil, &apperror.Err{
				Public:  true,
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
				Public:  true,
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
				Public:  true,
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
				Public:  true,
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

	// Add filters.
	if context.Has("filters") {
		parts := strings.Split(context.String("filters"), ",")
		for _, filter := range parts {
			filterParts := strings.Split(filter, ":")

			if len(filterParts) != 2 {
				return nil, &apperror.Err{
					Public:  true,
					Code:    "invalid_filter",
					Message: fmt.Sprintf("Invalid filter: %v", filter),
				}
			}

			fieldName := filterParts[0]

			// COnvert id query to pk field.
			if fieldName == "id" {
				fieldName = info.PkField
			}

			if !info.HasField(fieldName) {
				fieldName = info.MapMarshalName(fieldName)
			}

			if !info.HasField(fieldName) {
				return nil, &apperror.Err{
					Code:    "filter_for_inexistant_field",
					Message: fmt.Sprintf("Tried to filter with inexistant field %v", fieldName),
				}
			}

			fieldInfo := info.GetField(fieldName)

			if fieldInfo.IsRelation() {
				if fieldInfo.HasOne {
					fieldInfo = info.GetField(fieldInfo.HasOneField)
					fieldName = fieldInfo.Name
				} else {
					return nil, &apperror.Err{
						Public:  true,
						Code:    "cant_filter_on_relation",
						Message: fmt.Sprintf("Tried to filter on relationship field %v (only possible for has-one relations)", fieldName),
					}
				}
			}

			convertedVal, err := db.Convert(filterParts[1], fieldInfo.Type)
			if err != nil {
				return nil, &apperror.Err{
					Public: true,
					Code:   "inconvertible_filter_value",
					Message: fmt.Sprintf("Coult not convert filter value %v for field %v (should be %v)",
						filterParts[1], fieldName, fieldInfo.Type),
				}
			}

			query.Filter(fieldName, convertedVal)
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

	response := res.ApiUpdate(model, request)

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
