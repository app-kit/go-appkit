package jsonapi

import (
	"encoding/json"
	"fmt"
	"reflect"

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
	Relationships map[string]map[string][]*ApiModel `json:"relationships,omitempty"`
}

func (d *ApiModel) AddRelation(name string, data *ApiModel) {
	if d.Relationships == nil {
		d.Relationships = make(map[string]map[string][]*ApiModel)
	}

	if _, ok := d.Relationships[name]; !ok {
		d.Relationships[name] = make(map[string][]*ApiModel)
		d.Relationships[name]["data"] = make([]*ApiModel, 0)
	}

	d.Relationships[name]["data"] = append(d.Relationships[name]["data"], data)
}

func (d *ApiModel) GetRelation(name string) []*ApiModel {
	if _, ok := d.Relationships[name]; !ok {
		return nil
	}
	return d.Relationships[name]["data"]
}

func BuildModel(backend db.Backend, collection string, rawData []byte) (kit.Model, kit.Error) {
	var request ApiModelData
	if err := json.Unmarshal(rawData, &request); err != nil {
		return nil, kit.WrapError(err, "invalid_json_body", "")
	}

	if request.Data == nil {
		return nil, kit.AppError{Code: "no_model_data"}
	}

	data := request.Data
	if collection != "" {
		data.Type = collection
	}

	if data.Type == "" {
		return nil, kit.AppError{Code: "missing_model_type"}
	}

	if !backend.HasCollection(data.Type) {
		return nil, kit.AppError{
			Code:    "unknown_model_type",
			Message: fmt.Sprintf("The model type %v is not supported", data.Type),
		}
	}

	info := backend.ModelInfo(data.Type)

	rawModel, _ := backend.CreateModel(data.Type)
	model := rawModel.(kit.Model)

	if data.Id != "" {
		model.SetID(data.Id)
	}

	fieldData := make(map[string]interface{})
	for key := range data.Attributes {
		fieldName := info.MapMarshalName(key)
		if fieldName == "" {
			return nil, kit.AppError{
				Code:    "invalid_attribute",
				Message: fmt.Sprintf("The collection '%v' does not have a field '%v'", data.Type, key),
			}
		}

		fieldData[fieldName] = data.Attributes[key]
	}

	if err := db.UpdateModelFromData(info, model, fieldData); err != nil {
		return nil, kit.WrapError(err, "update_model_from_dict_error", "")
	}

	return model, nil
}

func ConvertModel(backend db.Backend, m kit.Model) (*ApiModel, []*ApiModel, kit.Error) {
	modelData, err := backend.ModelToMap(m, true)
	if err != nil {
		return nil, nil, kit.WrapError(err, "model_convert_error", "")
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
			return nil, nil, kit.WrapError(err, "model_get_field_error", "")
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
				return nil, nil, kit.WrapError(err, "included_model_convert_error", "")
			}

			// Build relation info and set in in relationships map.
			relation := &ApiModel{
				Type: relatedModel.Collection(),
				Id:   relatedModel.GetStrID(),
			}

			data.AddRelation(field.MarshalName, relation)

			// Add related model to included data.
			includedModels = append(includedModels, relationData)

			// Add nested included models to included data.
			includedModels = append(includedModels, included...)
		}
	}

	return data, includedModels, nil
}

func ConvertModels(backend db.Backend, models []kit.Model) ([]*ApiModel, []*ApiModel, kit.Error) {
	modelsData := make([]*ApiModel, 0)
	includedModels := make([]*ApiModel, 0)

	for _, m := range models {
		modelData, included, err := ConvertModel(backend, m)
		if err != nil {
			return nil, nil, kit.WrapError(err, "model_convert_error", "")
		}

		modelsData = append(modelsData, modelData)
		includedModels = append(includedModels, included...)
	}

	return modelsData, includedModels, nil
}

func ConvertError(err error) []*ApiError {
	errs := make([]*ApiError, 0)

	if appError, ok := err.(kit.Error); ok {
		if appError.IsInternal() {
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

	if data := resp.GetData(); data != nil {
		if model, ok := data.(kit.Model); ok {
			modelData, included, err := ConvertModel(backend, model)
			if err != nil {
				return ConvertResponse(backend, &kit.AppResponse{
					Error: err,
				})
			}

			apiResponse.Data = modelData
			apiResponse.Included = included
		} else if models, ok := data.([]kit.Model); ok {
			modelData, included, err := ConvertModels(backend, models)
			if err != nil {
				return ConvertResponse(backend, &kit.AppResponse{
					Error: err,
				})
			}

			apiResponse.Data = modelData
			apiResponse.Included = included
		} else {
			apiResponse.Data = data
		}
	}

	apiResponse.Meta = resp.GetMeta()

	js, err := json.Marshal(apiResponse)
	if err != nil {
		return ConvertResponse(backend, &kit.AppResponse{
			Error: kit.WrapError(err, "json_marshal_error", ""),
		})
	}

	return &kit.AppResponse{
		HttpStatus: resp.GetHttpStatus(),
		Error:      resp.GetError(),
		RawData:    js,
	}
}
