package appkit

import (
	db "github.com/theduke/dukedb"	
)

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
 * Resource.
 */

 type ApiResource interface {
 	GetBackend() db.Backend
 	SetBackend(db.Backend)

 	SetHooks(ApiHooks)

 	GetDebug() bool
 	SetDebug(bool)

 	SetUserHandler(ApiUserHandler)
	GetUserHandler() ApiUserHandler

 	GetModel() db.Model
 	SetModel(db.Model)

 	Query(*db.Query) ([]db.Model, ApiError)
 	GetQuery() *db.Query

 	FindOne(id string) (db.Model, ApiError)
 	ApiFindOne(id string, r ApiRequest) ApiResponse

 	Find(*db.Query) ([]db.Model, ApiError)
 	ApiFind(*db.Query, ApiRequest) ApiResponse

 	Create(obj db.Model, user ApiUser) ApiError
 	ApiCreate(obj db.Model, r ApiRequest) ApiResponse

 	Update(obj db.Model, user ApiUser) ApiError
 	ApiUpdate(obj db.Model, r ApiRequest) ApiResponse

 	Delete(obj db.Model, user ApiUser) ApiError
 	ApiDelete(id string, r ApiRequest) ApiResponse
}

type ApiHooks interface {

}

/**
 * Query hook.
 */

type UserCanQueryHook interface {
	UserCanQuery(q db.Query, user ApiUser)
}

/**
 * FindOne hooks.
 */

type ApiFindOneHook interface {
	ApiFindOne(res ApiResource, id string, r ApiRequest) ApiResponse
}

type UserCanFindOneHook interface {
	UserCanFindOne(res ApiResource, obj db.Model, user ApiUser) bool
}

type ApiAfterFindOneHook interface {
	ApiAfterFindOne(res ApiResource, obj db.Model, user ApiUser) ApiError
}

/**
 * Find hooks.
 */

type ApiFindHook interface {
	ApiFind(res ApiResource, query *db.Query, r ApiRequest) ApiResponse
}

type UserCanFindHook interface {
	UserCanFind(res ApiResource, objs []db.Model, user ApiUser) bool
}

type ApiAfterFindHook interface {
	ApiAfterFind(res ApiResource, objs []db.Model, user ApiUser) ApiError
}

/**
 * Create hooks.
 */

type ApiCreateHook interface {
	ApiCreate(res ApiResource, obj db.Model, r ApiRequest) ApiResponse
}

type CreateHook interface {
	Create(res ApiResource, obj db.Model, user ApiUser) ApiError
}

type BeforeCreateHook interface {
	BeforeCreate(res ApiResource, obj db.Model, user ApiUser) ApiError
}

type UserCanCreateHook interface {
	UserCanCreate(res ApiResource, obj db.Model, user ApiUser) bool
}

type AfterCreateHook interface {
	AfterCreate(res ApiResource, obj db.Model, user ApiUser) ApiError
}


/**
 * Update hooks.
 */

type ApiUpdateHook interface {
	ApiUpdate(res ApiResource, obj db.Model, r ApiRequest) ApiResponse
}

type UpdateHook interface {
	Update(res ApiResource, obj db.Model, r ApiRequest) ApiError
}


type BeforeUpdateHook interface {
	BeforeUpdate(res ApiResource, obj, oldobj db.Model, user ApiUser) ApiError
}

type AfterUpdateHook interface {
	AfterUpdate(res ApiResource, obj, oldobj db.Model, user ApiUser) ApiError
}

type UserCanUpdateHook interface {
	UserCanUpdate(res ApiResource, obj db.Model, old db.Model, user ApiUser) bool
}


/**
 * Delete hooks.
 */

type ApiDeleteHook interface {
	ApiDelete(res ApiResource, obj db.Model, r ApiRequest) ApiResponse
}

type DeleteHook interface {
	Delete(res ApiResource, obj db.Model, user ApiUser) ApiError
}

type BeforeDeleteHook interface {
	BeforeDelete(res ApiResource, obj db.Model, user ApiUser) ApiError
}

type UserCanDeleteHook interface {
	UserCanDelete(res ApiResource, obj db.Model, user ApiUser) bool
}

type AfterDeleteHook interface {
	AfterDelete(res ApiResource, obj db.Model, user ApiUser) ApiError
}

