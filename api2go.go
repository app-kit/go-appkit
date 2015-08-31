package appkit

import (
	"reflect"
	"strings"
	"errors"

	"github.com/manyminds/api2go/jsonapi"

	db "github.com/theduke/go-dukedb"
)

type Api2GoModelInterface interface {
	SetModelInfo(*db.ModelInfo)
	SetFullModel(db.Model)
}

type Api2GoModel struct {
	modelInfo *db.ModelInfo
	model db.Model
}

func (m *Api2GoModel) SetModelInfo(info *db.ModelInfo) {
	m.modelInfo = info
}

func (m *Api2GoModel) SetFullModel(model db.Model) {
	m.model = model
}

func (m Api2GoModel) GetReferences() []jsonapi.Reference {
	refs := make([]jsonapi.Reference, 0)

	for key := range m.modelInfo.FieldInfo {
		fieldInfo := m.modelInfo.FieldInfo[key]

		if fieldInfo.RelationItem != nil {
			refs = append(refs, jsonapi.Reference{
				Type: fieldInfo.RelationItem.Collection(),
				Name: fieldInfo.Name,
			})
		}
	}
	
	return refs
}

// GetReferencedIDs to satisfy the jsonapi.MarshalLinkedRelations interface
func (m Api2GoModel) GetReferencedIDs() []jsonapi.ReferenceID {
	result := make([]jsonapi.ReferenceID, 0)

	modelVal := reflect.ValueOf(m.model).Elem()
	modelType := modelVal.Type()

	for key := range m.modelInfo.FieldInfo {
		fieldInfo := m.modelInfo.FieldInfo[key]
		if fieldInfo.RelationItem == nil {
			continue
		}

		fieldVal := modelVal.FieldByName(key)
		tmp, _ := modelType.FieldByName(key)
		fieldType := tmp.Type

		if fieldType.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
			fieldType = fieldType.Elem()
		}

		if fieldInfo.RelationIsMany {
			for i := 0; i < fieldVal.Len(); i++ {
				sliceItem := fieldVal.Index(i)
				if sliceItem.Type().Kind() == reflect.Ptr {
					sliceItem = sliceItem.Elem()
				}

				model := sliceItem.Interface().(db.Model)
				result = append(result, jsonapi.ReferenceID{
					ID:   model.GetID(),
					Type: model.Collection(),
					Name: fieldInfo.Name,
				})
			}
		} else {
			model := fieldVal.Interface().(db.Model)
			result = append(result, jsonapi.ReferenceID{
				ID:   model.GetID(),
				Type: model.Collection(),
				Name: fieldInfo.Name,
			})
		}
	}

	return result
}

// GetReferencedStructs to satisfy the jsonapi.MarhsalIncludedRelations interface
func (m Api2GoModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := make([]jsonapi.MarshalIdentifier, 0)

	modelVal := reflect.ValueOf(m.model).Elem()
	modelType := modelVal.Type()

	for key := range m.modelInfo.FieldInfo {
		fieldInfo := m.modelInfo.FieldInfo[key]
		if fieldInfo.RelationItem == nil {
			continue
		}

		fieldVal := modelVal.FieldByName(key)
		tmp, _ := modelType.FieldByName(key)
		fieldType := tmp.Type

		if fieldType.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
			fieldType = fieldType.Elem()
		}

		if fieldInfo.RelationIsMany {
			for i := 0; i < fieldVal.Len(); i++ {
				sliceItem := fieldVal.Index(i)
				if sliceItem.Type().Kind() == reflect.Ptr {
					sliceItem = sliceItem.Elem()
				}

				result = append(result, sliceItem.Interface().(db.Model))
			}
		} else {
			model := fieldVal.Interface().(db.Model)
			result = append(result, model)
		}
	}

	return result
}

func (m *Api2GoModel) SetToOneReferenceID(name, ID string) error {
	name = strings.Replace(name, "-", "_", -1)
	fieldName := m.modelInfo.MapFieldName(name)

	if fieldName == "" {
		return errors.New("Unknown relation " + name)
	}

	fieldInfo := m.modelInfo.FieldInfo[fieldName]	

	if fieldInfo.RelationItem == nil {
		return errors.New(name + " is not a relationship")
	}

	if !fieldInfo.HasOne {
		return errors.New("Cannot set BelongsTo relationship " + name)
	}

	err := db.SetStructFieldValueFromString(m.model, fieldName, ID)
	if err != nil {
		return err
	}

	return nil
}
