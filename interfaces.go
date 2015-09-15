package appkit

import (
	"net/http"

	db "github.com/theduke/go-dukedb"

	. "github.com/theduke/go-appkit/error"
)

/**
 * API interface.
 */

type ApiRequest interface {
	GetUser() ApiUser
	SetUser(ApiUser)

	GetSession() ApiSession
	SetSession(ApiSession)

	GetContext() *Context
	GetMeta() Context
	GetData() interface{}

	GetHttpRequest() *http.Request
}

type ApiResponse interface {
	GetError() Error
	GetMeta() map[string]interface{}
	SetMeta(map[string]interface{})
	GetData() interface{}
}

/**
 * Resource.
 */

type ApiResource interface {
	App() *App
	SetApp(*App)

	GetBackend() db.Backend
	SetBackend(db.Backend)

	Hooks() ApiHooks
	SetHooks(ApiHooks)

	GetDebug() bool
	SetDebug(bool)

	SetUserHandler(ApiUserHandler)
	GetUserHandler() ApiUserHandler

	GetModel() db.Model
	SetModel(db.Model)
	NewModel() db.Model

	Q() *db.Query

	Find(*db.Query) ([]db.Model, Error)
	FindOne(id string) (db.Model, Error)

	ApiFindOne(string, ApiRequest) ApiResponse
	ApiFind(*db.Query, ApiRequest) ApiResponse
	// Same as find, but response meta will contain a total count.
	ApiFindPaginated(*db.Query, ApiRequest) ApiResponse

	Create(obj db.Model, user ApiUser) Error
	ApiCreate(obj db.Model, r ApiRequest) ApiResponse

	Update(obj db.Model, user ApiUser) Error
	ApiUpdate(obj db.Model, r ApiRequest) ApiResponse

	Delete(obj db.Model, user ApiUser) Error
	ApiDelete(id string, r ApiRequest) ApiResponse
}

type ApiHooks interface {
}

type ApiWithApp interface {
	SetApp(*App)
}

// Allow resource hooks to specify custom http routes.
type ApiHttpRoutes interface {
	// Allows to set up custom http handlers with the httprouter directly.
	HttpRoutes(ApiResource) []*HttpRoute
}

/**
 * Find hooks.
 */

type AllowFindHook interface {
	AllowFind(res ApiResource, model db.Model, user ApiUser) bool
}

type ApiFindHook interface {
	ApiFind(res ApiResource, query *db.Query, r ApiRequest) ApiResponse
}

type ApiAlterQueryHook interface {
	ApiAlterQuery(res ApiResource, query *db.Query, r ApiRequest) Error
}

type ApiAfterFindHook interface {
	ApiAfterFind(res ApiResource, obj []db.Model, user ApiUser) Error
}

/**
 * Create hooks.
 */

type ApiCreateHook interface {
	ApiCreate(res ApiResource, obj db.Model, r ApiRequest) ApiResponse
}

type CreateHook interface {
	Create(res ApiResource, obj db.Model, user ApiUser) Error
}

type BeforeCreateHook interface {
	BeforeCreate(res ApiResource, obj db.Model, user ApiUser) Error
}

type AllowCreateHook interface {
	AllowCreate(res ApiResource, obj db.Model, user ApiUser) bool
}

type AfterCreateHook interface {
	AfterCreate(res ApiResource, obj db.Model, user ApiUser) Error
}

/**
 * Update hooks.
 */

type ApiUpdateHook interface {
	ApiUpdate(res ApiResource, obj db.Model, r ApiRequest) ApiResponse
}

type UpdateHook interface {
	Update(res ApiResource, obj db.Model, r ApiRequest) Error
}

type BeforeUpdateHook interface {
	BeforeUpdate(res ApiResource, obj, oldobj db.Model, user ApiUser) Error
}

type AllowUpdateHook interface {
	AllowUpdate(res ApiResource, obj db.Model, old db.Model, user ApiUser) bool
}

type AfterUpdateHook interface {
	AfterUpdate(res ApiResource, obj, oldobj db.Model, user ApiUser) Error
}

/**
 * Delete hooks.
 */

type ApiDeleteHook interface {
	ApiDelete(res ApiResource, id string, r ApiRequest) ApiResponse
}

type DeleteHook interface {
	Delete(res ApiResource, obj db.Model, user ApiUser) Error
}

type BeforeDeleteHook interface {
	BeforeDelete(res ApiResource, obj db.Model, user ApiUser) Error
}

type AllowDeleteHook interface {
	AllowDelete(res ApiResource, obj db.Model, user ApiUser) bool
}

type AfterDeleteHook interface {
	AfterDelete(res ApiResource, obj db.Model, user ApiUser) Error
}
