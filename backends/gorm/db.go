package gorm

import (
	"strconv"
	"fmt"
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/theduke/appkit"
)

type Query struct {
	Run func(target interface{}, db *gorm.DB) appkit.ApiError

	Model string
	Limit int
	Offset int
}

func (q Query) GetModel() string {
	return q.Model
}

func (q *Query) SetModel(m string) {
	q.Model = m
}

func (q Query) GetLimit() int {
	return q.Limit
}

func (q *Query) SetLimit(m int) {
	q.Limit = m
}

func (q Query) GetOffset() int {
	return q.Offset
}

func (q *Query) SetOffset(m int) {
	q.Offset = m
}

type Backend struct {
	Db *gorm.DB
	typeMap map[string]interface{}
}

func interfaceToModelSlice(slicePtr interface{}) []appkit.ApiModel {
	reflSlice := reflect.ValueOf(slicePtr).Elem()
	result := make([]appkit.ApiModel, 0)

	for i := 0; i < reflSlice.Len(); i ++ {
		item := reflSlice.Index(i).Interface()
		result = append(result, item.(appkit.ApiModel))
	}

	return result
}

func NewBackend(url string) (*Backend, appkit.ApiError) {
	// connect to db
	db, err := gorm.Open("postgres", url)
	if err != nil {
		return nil, appkit.Error{
			Code: "db_connect_failed", 
			Message: fmt.Sprintf("Could not connect to %v: %v", url, err),
		}
	}

	b := Backend{}
	b.Db = &db
	b.typeMap = make(map[string]interface{})

	return &b, nil
}

func (b Backend) GetName() string {
	return "gorm"
}

func (b *Backend) RegisterModel(m appkit.ApiModel) {
	b.typeMap[m.GetName()] = interface{}(m)
}

func (b Backend) GetType(name string) (interface{}, appkit.ApiError) {
	typ, ok := b.typeMap[name]
	if !ok {
		return nil, appkit.Error{
			Code: "model_type_not_found", 
			Message: fmt.Sprintf("Model type '%v' not registered with backend GORM", name),
		}
	}

	// Build new struct.

	item := reflect.ValueOf(typ).Elem().Interface()
	return reflect.New(reflect.TypeOf(item)).Interface(), nil
}

func (b Backend) GetTypeSlice(name string) (interface{}, appkit.ApiError) {
	typ, ok := b.typeMap[name]
	if !ok {
		return nil, appkit.Error{
			Code: "model_type_not_found", 
			Message: fmt.Sprintf("Model type '%v' not registered with backend GORM", name),
		}
	}

	// Build new array.
	// See http://stackoverflow.com/questions/25384640/why-golang-reflect-makeslice-returns-un-addressable-value
	
	// Create a slice to begin with
	myType := reflect.TypeOf(typ)
	slice := reflect.MakeSlice(reflect.SliceOf(myType), 0, 99999)

	// Create a pointer to a slice value and set it to the slice
	x := reflect.New(slice.Type())
	x.Elem().Set(slice)

	slicePointer := x.Interface()

	return slicePointer, nil
}

func (b Backend) BuildQuery(appkit.RawQuery) (appkit.Query, appkit.ApiError) {
	return nil, appkit.Error{Code: "build_query_not_implemented"}
}

func (b Backend) Find(rawQuery appkit.Query) ([]appkit.ApiModel, appkit.ApiError) {
	slice, err := b.GetTypeSlice(rawQuery.GetModel())
	if err != nil {
		return nil, err
	}

	q, ok := rawQuery.(*Query)
	if !ok {
		return nil, appkit.Error{Code: "invalid_gorm_query"}
	}

	if q.Run != nil {
		// Custom run function.
		if err := q.Run(slice, b.Db); err != nil {
			return nil, appkit.Error{Code: "db_error", Message: err.Error()}
		}
	} else {
		if err := b.Db.Find(slice).Error; err != nil {
			return nil, appkit.Error{Code: "db_error", Message: err.Error()}
		}
	}

	return interfaceToModelSlice(slice), nil
}

func (b Backend) FindBy(modelType string, filters map[string]interface{}) ([]appkit.ApiModel, appkit.ApiError) {
	slice, err := b.GetTypeSlice(modelType)
	if err != nil {
		return nil, err
	}

	q := b.Db

	reflFilters := reflect.ValueOf(filters)

	for _, field := range reflFilters.MapKeys() {
		name := field.String()

		val := reflFilters.MapIndex(field).Interface()
		kind := reflFilters.MapIndex(field).Elem().Kind()

		// Fix all ID fields to be int instead of string.
		if strings.HasSuffix(name, "ID") && kind == reflect.String {
			id, err := strconv.ParseUint(val.(string), 10, 64)
			if err != nil {
				return nil, appkit.Error{
					Code: "invalid_id_filter",
					Message: fmt.Sprintf("Filter for field %v is not int", name),
				}
			}
			q = q.Where(name + " = ?", id)
		} else {
			q = q.Where(name + " = ?", val)
		}
	}

	if err := q.Find(slice).Error; err != nil {
		return nil, appkit.Error{Code: "db_error", Message: err.Error()}
	}

	return interfaceToModelSlice(slice), nil
}

func (b Backend) FindOneBy(modelType string, filters map[string]interface{}) (appkit.ApiModel, appkit.ApiError) {
	result, err := b.FindBy(modelType, filters)
	if err != nil {
		return nil, err
	}

	if len(result) < 1 {
		return nil, nil
	}

	return result[0], nil
}


func (b Backend) FindOne(modelType, rawId string) (appkit.ApiModel, appkit.ApiError) {
	obj, err := b.GetType(modelType)
	if err != nil {
		return nil, err
	}

	id, err2 := strconv.ParseUint(rawId, 10, 64)
	if err2 != nil {
		return nil, appkit.Error{Code: "invalid_id", Message: rawId}
	}

	if err := b.Db.Where("id = ?", id).First(obj).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, appkit.Error{Code: "db_error", Message: err.Error()}
	}

	m := obj.(appkit.ApiModel)

	return m, nil
}

func (b Backend) Create(m appkit.ApiModel) appkit.ApiError {
	if err := b.Db.Create(m).Error; err != nil {
		return appkit.Error{Code: "gorm_error", Message: err.Error()}
	}

	return nil
}

func (b Backend) Update(m appkit.ApiModel) appkit.ApiError {
	if err := b.Db.Save(m).Error; err != nil {
		return appkit.Error{Code: "gorm_error", Message: err.Error()}
	}

	return nil
}

func (b Backend) Delete(m appkit.ApiModel) appkit.ApiError {
	if err := b.Db.Delete(m).Error; err != nil {
		return appkit.Error{Code: "gorm_error", Message: err.Error()}
	}

	return nil
}
