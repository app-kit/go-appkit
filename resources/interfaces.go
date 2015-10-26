package resources

import (
	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
)

/**
 * Resource.
 */

// ApiHttpRoutes allows a reseource to specify custom http routes.
type ApiHttpRoutes interface {
	// Allows to set up custom http handlers with the httprouter directly.
	HttpRoutes(kit.Resource) []kit.HttpRoute
}

// MethodsHook allows a resource to specify additional methods.
type MethodsHook interface {
	Methods(kit.Resource) []kit.Method
}

/**
 * Find hooks.
 */

type AllowFindHook interface {
	AllowFind(res kit.Resource, model kit.Model, user kit.User) bool
}

type ApiFindOneHook interface {
	ApiFindOne(res kit.Resource, rawId string, r kit.Request) kit.Response
}

type ApiFindHook interface {
	ApiFind(res kit.Resource, query db.Query, r kit.Request) kit.Response
}

type ApiAlterQueryHook interface {
	// Alter an API query before it is executed.
	ApiAlterQuery(res kit.Resource, query db.Query, r kit.Request) apperror.Error
}

type ApiAfterFindHook interface {
	ApiAfterFind(res kit.Resource, objects []kit.Model, req kit.Request, resp kit.Response) apperror.Error
}

/**
 * Create hooks.
 */

type ApiCreateHook interface {
	ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
}

type CreateHook interface {
	Create(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

type BeforeCreateHook interface {
	BeforeCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

type AllowCreateHook interface {
	AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool
}

type AfterCreateHook interface {
	AfterCreate(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

/**
 * Update hooks.
 */

type ApiUpdateHook interface {
	ApiUpdate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response
}

type UpdateHook interface {
	Update(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

type BeforeUpdateHook interface {
	BeforeUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
}

type AllowUpdateHook interface {
	AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool
}

type AfterUpdateHook interface {
	AfterUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error
}

/**
 * Delete hooks.
 */

type ApiDeleteHook interface {
	ApiDelete(res kit.Resource, id string, r kit.Request) kit.Response
}

type DeleteHook interface {
	Delete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

type BeforeDeleteHook interface {
	BeforeDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}

type AllowDeleteHook interface {
	AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool
}

type AfterDeleteHook interface {
	AfterDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error
}
