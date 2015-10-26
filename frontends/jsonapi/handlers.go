package jsonapi

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
)

func HandleOptions(registry kit.Registry, r kit.Request) (kit.Response, bool) {
	return &kit.AppResponse{}, false
}

func HandleWrap(collection string, handler kit.RequestHandler) kit.RequestHandler {
	return func(registry kit.Registry, r kit.Request) (kit.Response, bool) {
		r.GetContext().Set("collection", strings.Replace(collection, "-", "_", -1))
		return handler(registry, r)
	}
}

func Find(res kit.Resource, request kit.Request) (kit.Response, apperror.Error) {
	collection := res.Collection()

	info := res.Backend().ModelInfo(collection)

	var query db.Query

	jsonQuery := request.GetContext().String("query")
	if jsonQuery != "" {
		var rawQuery map[string]interface{}
		if err := json.Unmarshal([]byte(jsonQuery), &rawQuery); err != nil {
			return nil, apperror.Wrap(err, "invalid_query_json")
		}

		rawQuery["collection"] = collection

		// A custom query was supplied.
		// Try to parse the query.
		var err apperror.Error
		query, err = db.ParseQuery(rawQuery)
		if err != nil {
			return nil, apperror.Wrap(err, "invalid_query", "", false)
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

func HandleFind(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")

	res := registry.Resource(collection)
	if res == nil || !res.IsPublic() {
		err := &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
		return kit.NewErrorResponse(err), false
	}

	response, err := Find(res, request)
	if err != nil {
		response = kit.NewErrorResponse(err)
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

	return response, false
}

func HandleFindOne(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")
	id := request.GetContext().MustString("id")

	res := registry.Resource(collection)
	if res == nil || !res.IsPublic() {
		resp := kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The resource '%v' does not exist", collection))
		return resp, false
	}

	return res.ApiFindOne(id, request), false
}

func Create(registry kit.Registry, request kit.Request) (kit.Response, apperror.Error) {
	collection := request.GetContext().MustString("collection")

	res := registry.Resource(collection)
	if res == nil || !res.IsPublic() {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
	}

	fmt.Printf("data: %v |  %+v\n\n", nil, request.GetData())
	model, ok := request.GetData().(kit.Model)
	if !ok {
		return nil, apperror.New("invalid_data_no_model", "No model data in request.")
	}

	response := res.ApiCreate(model, request)
	if response.GetError() == nil {
		response.SetHttpStatus(201)
	}

	return response, nil
}

func HandleCreate(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	response, err := Create(registry, request)
	if err != nil {
		return kit.NewErrorResponse(err), false
	}

	return response, false
}

func Update(registry kit.Registry, request kit.Request) (kit.Response, apperror.Error) {
	collection := request.GetContext().MustString("collection")

	res := registry.Resource(collection)
	if res == nil || !res.IsPublic() {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource '%v' does not exist", collection),
		}
	}

	model, ok := request.GetData().(kit.Model)
	if !ok {
		return nil, apperror.New("invalid_data_no_model", "Node model data in request.")
	}

	response := res.ApiUpdate(model, request)

	return response, nil
}

func HandleUpdate(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	response, err := Update(registry, request)
	if err != nil {
		return kit.NewErrorResponse(err), false
	}

	return response, false
}

func HandleDelete(registry kit.Registry, request kit.Request) (kit.Response, bool) {
	collection := request.GetContext().MustString("collection")
	id := request.GetContext().MustString("id")

	res := registry.Resource(collection)
	if res == nil || !res.IsPublic() {
		resp := kit.NewErrorResponse("unknown_resource", fmt.Sprintf("The resource '%v' does not exist", collection))
		return resp, false
	}

	return res.ApiDelete(id, request), false
}
