package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type ApiError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type ApiData struct {
	Data     interface{}            `json:"data,omitempty"`
	Included []*ApiModel            `json:"included,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
	Errors   []*ApiError            `json:"errors,omitempty"`
}

type ApiModelData struct {
	ApiData
	Data *ApiModel `json:"data,omitempty"`
}

type ApiModelsData struct {
	ApiData
	Data []*ApiModel `json:"data,omitempty"`
}

type ApiModel struct {
	Type          string                            `json:"type"`
	Id            string                            `json:"id"`
	Attributes    map[string]interface{}            `json:"attributes,omitempty"`
	Relationships map[string]map[string]interface{} `json:"relationships,omitempty"`
}

func ApiModelFromMap(data map[string]interface{}) (*ApiModel, apperror.Error) {
	rawType, ok := data["type"]
	if !ok {
		return nil, &apperror.Err{Code: "invalid_data_no_type"}
	}

	typ, ok := rawType.(string)
	if !ok {
		return nil, &apperror.Err{Code: "invalid_data_type_not_a_string"}
	}

	rawId, ok := data["id"]
	if !ok {
		return nil, &apperror.Err{Code: "invalid_data_no_id"}
	}

	id, ok := rawId.(string)
	if !ok {
		return nil, &apperror.Err{Code: "invalid_data_id_not_a_string"}
	}

	return &ApiModel{
		Type: typ,
		Id:   id,
	}, nil
}

func ApiModelsFromData(data interface{}) ([]*ApiModel, apperror.Error) {
	if item, ok := data.(map[string]interface{}); ok {
		if model, err := ApiModelFromMap(item); err != nil {
			return nil, err
		} else {
			return []*ApiModel{model}, nil
		}
	}

	// Not a single model, so should be a slice.
	if slice, ok := data.([]interface{}); ok {
		models := make([]*ApiModel, 0)

		for _, rawItem := range slice {
			item := rawItem.(map[string]interface{})

			if model, err := ApiModelFromMap(item); err != nil {
				return nil, err
			} else {
				models = append(models, model)
			}
		}

		return models, nil
	}

	return nil, apperror.New("invalid_data")
}

func (d *ApiModel) AddRelation(name string, data *ApiModel, isSingle bool) {
	if d.Relationships == nil {
		d.Relationships = make(map[string]map[string]interface{})
	}

	if _, ok := d.Relationships[name]; !ok {
		d.Relationships[name] = make(map[string]interface{})

		if !isSingle {
			d.Relationships[name]["data"] = make([]*ApiModel, 0)
		}
	}

	if isSingle {
		d.Relationships[name]["data"] = data
	} else {
		d.Relationships[name]["data"] = append(d.Relationships[name]["data"].([]*ApiModel), data)
	}
}

func (d *ApiModel) GetRelation(name string) ([]*ApiModel, apperror.Error) {
	if _, ok := d.Relationships[name]; !ok {
		return nil, nil
	}
	data := d.Relationships[name]["data"]

	if item, ok := data.(*ApiModel); ok {
		return []*ApiModel{item}, nil
	} else if items, ok := data.([]*ApiModel); ok {
		return items, nil
	} else {
		return ApiModelsFromData(data)
	}
}

func (d ApiModel) GetRelations() (map[string][]*ApiModel, apperror.Error) {
	rels := make(map[string][]*ApiModel)

	if d.Relationships != nil {
		for key := range d.Relationships {
			models, err := d.GetRelation(key)
			if err != nil {
				return nil, err
			}
			if models != nil {
				rels[key] = models
			}
		}
	}

	return rels, nil
}

func BuildModel(backend db.Backend, collection string, rawData []byte) (kit.Model, apperror.Error) {
	var request ApiModelData
	if err := json.Unmarshal(rawData, &request); err != nil {
		return nil, apperror.Wrap(err, "invalid_json_body", true)
	}

	if request.Data == nil {
		return nil, apperror.New("no_model_data", true)
	}

	data := request.Data
	if collection != "" {
		data.Type = collection
	}

	if data.Type == "" {
		return nil, apperror.New("missing_model_type", true)
	}

	if !backend.HasCollection(data.Type) {
		return nil, &apperror.Err{
			Public:  true,
			Code:    "unknown_model_type",
			Message: fmt.Sprintf("The model type %v is not supported", data.Type, true),
		}
	}

	info := backend.ModelInfo(data.Type)

	rawModel, _ := backend.CreateModel(data.Type)
	model := rawModel.(kit.Model)

	fieldData := make(map[string]interface{})
	for key := range data.Attributes {
		fieldName := info.MapMarshalName(key)
		if fieldName == "" {
			return nil, &apperror.Err{
				Public:  true,
				Code:    "invalid_attribute",
				Message: fmt.Sprintf("The collection '%v' does not have a field '%v'", data.Type, key),
			}
		}

		fieldData[fieldName] = data.Attributes[key]
	}

	// Set ID if supplied.
	if data.Id != "" {
		if err := model.SetStrID(data.Id); err != nil {
			return nil, apperror.Wrap(err, "invalid_id", true)
		}
	}

	if err := db.UpdateModelFromData(info, model, fieldData); err != nil {
		return nil, apperror.Wrap(err, "update_model_from_dict_error", "")
	}

	// Now, try to handle relationships.
	allRelations, err := data.GetRelations()
	if err != nil {
		return nil, apperror.Wrap(err, "invalid_relationship_data")
	}

	for relationship, items := range allRelations {
		if len(items) < 1 {
			continue
		}

		if !info.HasField(relationship) {
			relationship = info.MapMarshalName(relationship)
		}

		if !info.HasField(relationship) {
			return nil, &apperror.Err{
				Public:  true,
				Code:    "invalid_relationship",
				Message: fmt.Sprintf("The collection %v does not have a relationship %v", collection, relationship),
			}
		}

		fieldInfo := info.GetField(relationship)
		relatedInfo := backend.ModelInfo(fieldInfo.RelationCollection)

		// Get a new related model for ID conversion.
		rawModel, err := backend.CreateModel(relatedInfo.Collection)
		if err != nil {
			return nil, apperror.Wrap(err, "create_related_model_error")
		}

		relatedModel := rawModel.(kit.Model)

		// Handle has-one field.
		if fieldInfo.HasOne {
			if len(items) != 1 {
				return nil, &apperror.Err{
					Code:    "multiple_items_for_has_one_relationship",
					Message: fmt.Sprintf("Data contains more than one item for has-one relationshiop %v", relationship),
				}
			}

			item := items[0]
			if item.Type != fieldInfo.RelationCollection {
				return nil, &apperror.Err{
					Public:  true,
					Code:    "invalid_relationship_type",
					Message: fmt.Sprintf("The item with id %v supplied for relationship %v has wrong type %v", item.Id, relationship, item.Type),
				}

				targetModel, err := backend.FindOne(fieldInfo.RelationCollection, item.Id)
				if err != nil {
					return nil, apperror.Wrap(err, "db_error", true)
				}

				err2 := db.SetStructModelField(model, fieldInfo.Name, []interface{}{targetModel})
				if err2 != nil {
					return nil, apperror.Wrap(err2, "assing_relationship_models_error")
				}
			}
		}

		// Handle m2m field.
		if fieldInfo.M2M {
			// First, collect the IDs of all related models.

			ids := make([]interface{}, 0)
			for _, item := range items {

				// Ensure that item has the correct collection.
				if item.Type != relatedInfo.Collection {
					return nil, &apperror.Err{
						Public:  true,
						Code:    "invalid_relationship_type",
						Message: fmt.Sprintf("The item with id %v supplied for relationship %v has wrong type %v", item.Id, relationship, item.Type),
					}
				}

				if item.Id == "" {
					return nil, &apperror.Err{
						Public:  true,
						Code:    "relationship_item_without_id",
						Message: fmt.Sprintf("An item for relationship %v does not have an id", relationship),
					}
				}

				// Use the related model to convert the id.
				if err := relatedModel.SetStrID(item.Id); err != nil {
					return nil, &apperror.Err{
						Public:  true,
						Code:    "invalid_relationship_item_id",
						Message: fmt.Sprintf("Item for relationship %v has invalid id %v", relationship, item.Id),
					}
				}

				ids = append(ids, relatedModel.GetID())
			}

			// Now, query the records from the database.
			res, err := backend.Q(relatedInfo.Collection).FilterCond(relatedInfo.PkField, "in", ids).Find()
			if err != nil {
				return nil, apperror.Wrap(err, "db_error")
			}

			if len(res) != len(ids) {
				return nil, &apperror.Err{
					Code:    "inexistant_relationship_ids",
					Message: fmt.Sprintf("Supplied non-existant ids for relationship %v", relationship),
				}
			}

			// Now we can update the model.
			if err := db.SetStructModelField(model, fieldInfo.Name, res); err != nil {
				return nil, apperror.Wrap(err, "assing_relationship_models_error")
			}
		}
	}

	return model, nil
}

func ConvertModel(backend db.Backend, m kit.Model) (*ApiModel, []*ApiModel, apperror.Error) {
	modelData, err := backend.ModelToMap(m, true)
	if err != nil {
		return nil, nil, apperror.Wrap(err, "model_convert_error", "")
	}

	info := backend.ModelInfo(m.Collection())

	data := &ApiModel{
		Type:       m.Collection(),
		Id:         m.GetStrID(),
		Attributes: modelData,
	}

	// Build relationship data.
	includedModels := make([]*ApiModel, 0)

	// Check every model  field.

	for fieldName := range info.FieldInfo {
		field := info.FieldInfo[fieldName]

		if !field.IsRelation() {
			// Not a relatinship field, so skip.
			continue
		}

		// Retrieve the related model.
		fieldVal, err := db.GetStructField(m, fieldName)
		if err != nil {
			return nil, nil, apperror.Wrap(err, "model_get_field_error", "")
		}

		// If field is zero value, skip.
		if db.IsZero(fieldVal.Interface()) {
			continue
		}

		related := make([]kit.Model, 0)

		if !field.RelationIsMany {
			// Make sure that we have a pointer.
			if fieldVal.Type().Kind() == reflect.Struct {
				fieldVal = fieldVal.Addr()
			}

			related = append(related, fieldVal.Interface().(kit.Model))
		} else {
			for i := 0; i < fieldVal.Len(); i++ {
				item := fieldVal.Index(i)
				if item.Type().Kind() == reflect.Struct {
					item = item.Addr()
				}

				related = append(related, item.Interface().(kit.Model))
			}
		}

		for _, relatedModel := range related {
			// Convert the related model.
			relationData, included, err := ConvertModel(backend, relatedModel)
			if err != nil {
				return nil, nil, apperror.Wrap(err, "included_model_convert_error", "")
			}

			// Build relation info and set in in relationships map.
			relation := &ApiModel{
				Type: relatedModel.Collection(),
				Id:   relatedModel.GetStrID(),
			}

			isSingle := !field.RelationIsMany
			data.AddRelation(field.MarshalName, relation, isSingle)

			// Add related model to included data.
			includedModels = append(includedModels, relationData)

			// Add nested included models to included data.
			includedModels = append(includedModels, included...)
		}
	}

	return data, includedModels, nil
}

func ConvertModels(backend db.Backend, models []kit.Model) ([]*ApiModel, []*ApiModel, apperror.Error) {
	modelsData := make([]*ApiModel, 0)
	includedModels := make([]*ApiModel, 0)

	for _, m := range models {
		modelData, included, err := ConvertModel(backend, m)
		if err != nil {
			return nil, nil, apperror.Wrap(err, "model_convert_error", "")
		}

		modelsData = append(modelsData, modelData)
		includedModels = append(includedModels, included...)
	}

	return modelsData, includedModels, nil
}

func ConvertError(err error) []*ApiError {
	errs := make([]*ApiError, 0)

	if appError, ok := err.(apperror.Error); ok {
		if !appError.IsPublic() {
			// Internal error, so do not provide any details.
			errs = append(errs, &ApiError{Code: "internal_server_error"})
		} else {
			// Not an internal error, show details.
			errs = append(errs, &ApiError{
				Code:    appError.GetCode(),
				Message: appError.GetMessage(),
			})
		}

		// Add any additional errors.
		for _, err := range appError.GetErrors() {
			errs = append(errs, ConvertError(err)...)
		}
	} else {
		errs = append(errs, &ApiError{Message: err.Error()})
	}

	return errs
}

func ConvertResponse(backend db.Backend, resp kit.Response) kit.Response {
	apiResponse := &ApiData{}

	if err := resp.GetError(); err != nil {
		apiResponse.Errors = ConvertError(err)
	}

	var modelData interface{}
	var included []*ApiModel
	var err apperror.Error

	if data := resp.GetData(); data != nil {
		if model, ok := data.(kit.Model); ok {
			modelData, included, err = ConvertModel(backend, model)
		} else if models, ok := data.([]kit.Model); ok {
			modelData, included, err = ConvertModels(backend, models)
		} else {
			modelData = data
		}
	}

	if err != nil {
		return ConvertResponse(backend, &kit.AppResponse{
			Error: err,
		})
	}

	meta := resp.GetMeta()

	// Check meta for modeldata to include.
	if meta != nil {
		for key, val := range meta {
			if model, ok := val.(kit.Model); ok {
				data, metaIncluded, err := ConvertModel(backend, model)

				if err != nil {
					return ConvertResponse(backend, &kit.AppResponse{
						Error: err,
					})
				}

				included = append(included, data)
				included = append(included, metaIncluded...)

				// Delete model from meta.
				delete(meta, key)
			}
		}

		// Set remaining meta data.
		apiResponse.Meta = meta
	}

	apiResponse.Data = modelData
	apiResponse.Included = included

	js, err2 := json.Marshal(apiResponse)
	if err2 != nil {
		return ConvertResponse(backend, &kit.AppResponse{
			Error: apperror.Wrap(err2, "json_marshal_error", ""),
		})
	}

	return &kit.AppResponse{
		HttpStatus: resp.GetHttpStatus(),
		Error:      resp.GetError(),
		RawData:    js,
	}
}
