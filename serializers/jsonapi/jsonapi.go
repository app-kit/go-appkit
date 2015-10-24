package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

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

func (d ApiData) ToMap() map[string]interface{} {
	// Todo: fix this ugly hack.

	js, _ := json.Marshal(d)
	var m map[string]interface{}

	json.Unmarshal(js, &m)

	return m
}

func (d *ApiData) ReduceIncludedDuplicates() {
	if len(d.Included) < 1 {
		return
	}

	cleanedModels := make([]*ApiModel, 0)
	mapper := make(map[string]bool)

	for _, model := range d.Included {
		id := model.Type + "_" + model.Id
		if _, ok := mapper[id]; !ok {
			cleanedModels = append(cleanedModels, model)
			mapper[id] = true
		}
	}

	d.Included = cleanedModels
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

func (d *ApiModel) GetRelation(name string) ([]*ApiModel, bool, apperror.Error) {
	if _, ok := d.Relationships[name]; !ok {
		return nil, false, nil
	}
	data := d.Relationships[name]["data"]

	if data == nil {
		return nil, false, nil
	}

	if item, ok := data.(*ApiModel); ok {
		return []*ApiModel{item}, false, nil
	} else if items, ok := data.([]*ApiModel); ok {
		return items, true, nil
	} else {
		return ApiModelsFromData(data)
	}
}

func (d ApiModel) GetRelations() (map[string][]*ApiModel, apperror.Error) {
	rels := make(map[string][]*ApiModel)

	if d.Relationships != nil {
		for key := range d.Relationships {
			models, _, err := d.GetRelation(key)
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

func ApiModelFromMap(rawData interface{}) (*ApiModel, apperror.Error) {
	data, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, apperror.New("invalid_data", "Invalid model data: dict expected", true)
	}

	model := &ApiModel{}

	// Build the type.
	typ, ok := data["type"].(string)
	if !ok || typ == "" {
		return nil, apperror.New("invalid_data_no_or_invalid_or_empty_type", true)
	}

	model.Type = typ

	// Find ID.
	rawId, ok := data["id"]
	if ok && rawId != nil {
		id, ok := rawId.(string)
		if !ok {
			return nil, apperror.New("invalid_data_id_not_a_string", true)
		}

		model.Id = id
	}

	// Attributes.
	if attrs, ok := data["attributes"].(map[string]interface{}); ok {
		model.Attributes = attrs
	}

	// Relationships.
	if rels, ok := data["relationships"].(map[string]interface{}); ok {
		relationships := make(map[string]map[string]interface{})

		for name, rawData := range rels {
			relModels, isMulti, err := ApiModelsFromData(rawData)
			if err != nil {
				return nil, apperror.Wrap(err, "invalid_relationship", fmt.Sprintf("Invalid relationship %v", name), true)
			}

			if isMulti {
				relationships[name] = map[string]interface{}{"data": relModels}
			} else {
				relationships[name] = map[string]interface{}{"data": relModels[0]}
			}
		}

		model.Relationships = relationships
	}

	return model, nil
}

func ApiModelsFromData(data interface{}) ([]*ApiModel, bool, apperror.Error) {
	if item, ok := data.(map[string]interface{}); ok {
		if model, err := ApiModelFromMap(item); err != nil {
			return nil, false, err
		} else {
			return []*ApiModel{model}, false, nil
		}
	}

	// Not a single model, so should be a slice.
	if slice, ok := data.([]interface{}); ok {
		models := make([]*ApiModel, 0)

		for _, itemData := range slice {
			if model, err := ApiModelFromMap(itemData); err != nil {
				return nil, true, err
			} else {
				models = append(models, model)
			}
		}

		return models, true, nil
	}

	return nil, false, apperror.New("invalid_data", true)
}

func SerializeModel(backend db.Backend, m kit.Model) (*ApiModel, []*ApiModel, apperror.Error) {
	modelData, err := backend.ModelToMap(m, true, false)
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
			return nil, nil, apperror.Wrap(err, "model_get_field_error")
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
			relationData, included, err := SerializeModel(backend, relatedModel)
			if err != nil {
				return nil, nil, apperror.Wrap(err, "included_model_serialize_error", "")
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

func SerializeModels(backend db.Backend, models []kit.Model) ([]*ApiModel, []*ApiModel, apperror.Error) {
	modelsData := make([]*ApiModel, 0)
	includedModels := make([]*ApiModel, 0)

	for _, m := range models {
		modelData, included, err := SerializeModel(backend, m)
		if err != nil {
			return nil, nil, apperror.Wrap(err, "model_convert_error", "")
		}

		modelsData = append(modelsData, modelData)
		includedModels = append(includedModels, included...)
	}

	return modelsData, includedModels, nil
}

func SerializeError(err error) []*ApiError {
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
			errs = append(errs, SerializeError(err)...)
		}
	} else {
		errs = append(errs, &ApiError{Message: err.Error()})
	}

	return errs
}

type Serializer struct {
	backends map[string]db.Backend
}

// Ensure Serializer implements kit.Serializer.
var _ kit.Serializer = (*Serializer)(nil)

func New(backends map[string]db.Backend) *Serializer {
	s := &Serializer{
		backends: backends,
	}

	return s
}

func (s *Serializer) Name() string {
	return "jsonapi"
}

func (s *Serializer) findBackend(collection string) db.Backend {
	for _, backend := range s.backends {
		if backend.HasCollection(collection) {
			return backend
		}
	}

	return nil
}

// SerializeModel converts a model into the target format.
func (s *Serializer) SerializeModel(model kit.Model) (interface{}, []interface{}, apperror.Error) {
	// Find backend.
	backend := s.findBackend(model.Collection())
	if backend == nil {
		return nil, nil, apperror.New("unknown_collection", fmt.Sprintf("Can't serialize model of unknown collection %v", model.Collection()))
	}

	// Serialize model.
	m, extra, err := SerializeModel(backend, model)
	if err != nil {
		return nil, nil, err
	}

	// Convert extra slice to interface slice.
	var rawExtra []interface{}
	for _, item := range extra {
		rawExtra = append(rawExtra, item)
	}

	return m, rawExtra, nil
}

func (s *Serializer) UnserializeModel(collection string, rawData interface{}) (kit.Model, apperror.Error) {
	// Fill in collection if it is not set in data.
	if data, ok := rawData.(map[string]interface{}); ok {
		if _, ok := data["type"].(string); !ok && collection != "" {
			data["type"] = collection
		}
	}

	data, err := ApiModelFromMap(rawData)
	if err != nil {
		return nil, err
	}

	// Replace dashes with underscore to support some JSONAPI implementations.
	data.Type = strings.Replace(data.Type, "-", "_", -1)

	backend := s.findBackend(data.Type)

	if backend == nil {
		return nil, &apperror.Err{
			Public:  true,
			Code:    "unknown_model_type",
			Message: fmt.Sprintf("The model type %v is not supported", data.Type),
		}
	}

	info := backend.ModelInfo(data.Type)

	var rawModel interface{}

	if data.Id != "" {
		model, err := backend.FindOne(data.Type, data.Id)
		if err != nil {
			return nil, err
		}
		if model == nil {
			return nil, &apperror.Err{
				Public:  true,
				Code:    "inexistant_model",
				Message: fmt.Sprintf("Model in collection %v with id %v does not exist", collection, data.Id),
			}
		}
		rawModel = model
	} else {
		rawModel, _ = backend.CreateModel(data.Type)
	}

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
		return nil, apperror.Wrap(err, "invalid_relationship_data", true)
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
			}

			targetModel, err := backend.FindOne(fieldInfo.RelationCollection, item.Id)
			if err != nil {
				return nil, apperror.Wrap(err, "db_error", true)
			}
			if targetModel == nil {
				return nil, &apperror.Err{
					Code: "inexistant_related_item",
					Message: fmt.Sprintf("Model for relationship %v (collection %v) with id %v does not exist",
						relationship, fieldInfo.RelationCollection, item.Id),
				}
			}

			foreignKey, _ := db.GetStructFieldValue(targetModel, fieldInfo.HasOneForeignField)
			err2 := db.SetStructField(model, fieldInfo.HasOneField, foreignKey)
			if err2 != nil {
				return nil, apperror.Wrap(err2, "assing_relationship_models_error")
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
					Public:  true,
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

// SerializeResponse converts a response with model data into the target format.
func (s *Serializer) SerializeResponse(response kit.Response) (interface{}, apperror.Error) {
	apiResponse := &ApiData{}

	if err := response.GetError(); err != nil {
		apiResponse.Errors = SerializeError(err)
	}

	var modelData interface{}
	var included []*ApiModel
	var err apperror.Error

	if data := response.GetData(); data != nil {
		if model, ok := data.(kit.Model); ok {
			// Find backend.
			backend := s.findBackend(model.Collection())
			if backend == nil {
				return nil, apperror.New("unknown_model_collection", fmt.Sprintf("Can't serialize unknown collection %v", model.Collection()))
			}

			// Serialize the model.
			modelData, included, err = SerializeModel(backend, model)
		} else if models, ok := data.([]kit.Model); ok {
			if len(models) > 0 {
				// Find backend.
				backend := s.findBackend(models[0].Collection())
				if backend == nil {
					return nil, apperror.New("unknown_model_collection", fmt.Sprintf("Can't serialize model of unknown collection %v", model.Collection()))
				}

				// Serialize the models.
				modelData, included, err = SerializeModels(backend, models)
			} else {
				modelData = make([]interface{}, 0)
			}
		} else {
			// Data is not a model or a slice of models, so just include the plain data.
			modelData = data
		}
	}

	if err != nil {
		return nil, err
	}

	meta := response.GetMeta()

	// Check meta for modeldata to include.
	if meta != nil {
		for key, val := range meta {
			if model, ok := val.(kit.Model); ok {
				backend := s.findBackend(model.Collection())
				if backend == nil {
					return nil, apperror.New("unknown_model_collection", fmt.Sprintf("Can't serialize model of unknown collection %v", model.Collection()))
				}

				data, metaIncluded, err := SerializeModel(backend, model)
				if err != nil {
					return nil, err
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

	// Remove duplicates from included data.
	apiResponse.ReduceIncludedDuplicates()

	return apiResponse.ToMap(), nil
}

func (s *Serializer) MustSerializeResponse(response kit.Response) interface{} {
	data, err := s.SerializeResponse(response)
	if err != nil {
		return &ApiData{
			Errors: SerializeError(err),
		}
	}

	return data
}

// UnserializeRequest converts request data into a request object.
func (s *Serializer) UnserializeRequest(rawData interface{}, request kit.Request) apperror.Error {
	if rawData == nil {
		return nil
	}

	allData, ok := rawData.(map[string]interface{})
	if !ok {
		return apperror.New("invalid_data", "Invalid request data: dict expected", true)
	}

	// Handle model data.
	if data, ok := allData["data"]; ok {
		// Handle map data.
		if mapData, ok := data.(map[string]interface{}); ok {
			// Check if data looks like model data.
			if _, ok := mapData["type"]; ok {
				model, err := s.UnserializeModel("", mapData)
				if err != nil {
					return err
				}
				request.SetData(model)
			} else {
				// Assume regular data.
				request.SetData(mapData)
			}
		} else if sliceData, ok := data.([]interface{}); ok {
			// Data looks like slice of models.
			models := make([]kit.Model, 0)

			for _, item := range sliceData {
				model, err := s.UnserializeModel("", item)
				if err != nil {
					return err
				}

				models = append(models, model)
			}

			request.SetData(models)
		}
	}

	// Handle metadata.
	if rawMeta, ok := allData["meta"]; ok && rawMeta != nil {
		meta, ok := rawMeta.(map[string]interface{})
		if !ok {
			return apperror.New("invalid_metadata", "Invalid metadata: dict expected", true)
		}

		request.SetMeta(kit.NewContext(meta))
	}

	return nil
}
