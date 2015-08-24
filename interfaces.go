package appkit

import (
	
)

type ApiConfig interface{

}

/**
 * Resource.
 */

 type ApiResource interface {
 	GetBackend() Backend
 	SetBackend(Backend)

 	GetDebug() bool
 	SetDebug(bool)

 	SetUserHandler(ApiUserHandler)
	GetUserHandler() ApiUserHandler

 	GetModel() ApiModel
 	SetModel(ApiModel)

	FindOneBy(map[string]interface{}) (ApiModel, ApiError)
 	FindOne(id string) (ApiModel, ApiError)
 	ApiFindOne(id string, r ApiRequest) ApiResponse

 	FindBy(map[string]interface{}) ([]ApiModel, ApiError)
 	Find(Query) ([]ApiModel, ApiError)
 	ApiFind(RawQuery, ApiRequest) ApiResponse

 	Create(obj ApiModel) ApiError
 	ApiCreate(obj ApiModel, r ApiRequest) ApiResponse

 	Update(obj ApiModel) ApiError
 	ApiUpdate(obj ApiModel, r ApiRequest) ApiResponse

 	Delete(obj ApiModel) ApiError
 	ApiDelete(id string, r ApiRequest) ApiResponse
}


/**
 * API interface.
 */

type ApiRequest interface {
	GetUser() ApiUser
	GetSession() ApiSession
	GetContext() Context
	GetMeta() Context
	GetData() interface{}
}

type ApiResponse interface {
	GetError() ApiError	
	GetMeta() map[string]interface{}
	GetData() interface{}
}

/**
 * DB interfaces.
 */

type ApiModel interface {
	GetName() string
	GetID() string
}

type Query interface {
	GetModel() string
	SetModel(string)

	GetLimit() int
	SetLimit(int)
	GetOffset() int
	SetOffset(int)
}

type Backend interface {
	GetName() string

	RegisterModel(ApiModel)

	// Get a new struct instance for a model type.
	GetType(string) (interface{}, ApiError)
	GetTypeSlice(string) (interface{}, ApiError)
	
	Find(Query) ([]ApiModel, ApiError)
	FindOne(modelType string, id string) (ApiModel, ApiError)

	FindBy(modelType string, filters map[string]interface{}) ([]ApiModel, ApiError)
	FindOneBy(modelType string, filters map[string]interface{}) (ApiModel, ApiError)

	BuildQuery(RawQuery) (Query, ApiError)

	Create(ApiModel) ApiError
	Update(ApiModel) ApiError
	Delete(ApiModel) ApiError
}

