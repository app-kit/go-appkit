package resources

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

/**
 * Resource.
 */

// Allow resource hooks to specify custom http routes.
type ApiHttpRoutes interface {
	// Allows to set up custom http handlers with the httprouter directly.
	HttpRoutes(kit.Resource) []kit.HttpRoute
}

type MethodsHook interface {
	Methods(kit.Resource) []kit.Method
}

/**
 * Find hooks.
 */

type AllowFindHook interface {
	AllowFind(res kit.Resource, model db.Model, user kit.User) bool
}

type ApiFindHook interface {
	ApiFind(res kit.Resource, query db.Query, r kit.Request) kit.Response
}

type ApiAlterQueryHook interface {
	ApiAlterQuery(res kit.Resource, query db.Query, r kit.Request) kit.Error
}

type ApiAfterFindHook interface {
	ApiAfterFind(res kit.Resource, obj []db.Model, user kit.User) kit.Error
}

/**
 * Create hooks.
 */

type ApiCreateHook interface {
	ApiCreate(res kit.Resource, obj db.Model, r kit.Request) kit.Response
}

type CreateHook interface {
	Create(res kit.Resource, obj db.Model, user kit.User) kit.Error
}

type BeforeCreateHook interface {
	BeforeCreate(res kit.Resource, obj db.Model, user kit.User) kit.Error
}

type AllowCreateHook interface {
	AllowCreate(res kit.Resource, obj db.Model, user kit.User) bool
}

type AfterCreateHook interface {
	AfterCreate(res kit.Resource, obj db.Model, user kit.User) kit.Error
}

/**
 * Update hooks.
 */

type ApiUpdateHook interface {
	ApiUpdate(res kit.Resource, obj db.Model, r kit.Request) kit.Response
}

type UpdateHook interface {
	Update(res kit.Resource, obj db.Model, r kit.Request) kit.Error
}

type BeforeUpdateHook interface {
	BeforeUpdate(res kit.Resource, obj, oldobj db.Model, user kit.User) kit.Error
}

type AllowUpdateHook interface {
	AllowUpdate(res kit.Resource, obj db.Model, old db.Model, user kit.User) bool
}

type AfterUpdateHook interface {
	AfterUpdate(res kit.Resource, obj, oldobj db.Model, user kit.User) kit.Error
}

/**
 * Delete hooks.
 */

type ApiDeleteHook interface {
	ApiDelete(res kit.Resource, id string, r kit.Request) kit.Response
}

type DeleteHook interface {
	Delete(res kit.Resource, obj db.Model, user kit.User) kit.Error
}

type BeforeDeleteHook interface {
	BeforeDelete(res kit.Resource, obj db.Model, user kit.User) kit.Error
}

type AllowDeleteHook interface {
	AllowDelete(res kit.Resource, obj db.Model, user kit.User) bool
}

type AfterDeleteHook interface {
	AfterDelete(res kit.Resource, obj db.Model, user kit.User) kit.Error
}
