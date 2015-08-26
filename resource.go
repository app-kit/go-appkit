package appkit

import (
	"reflect"
	//"log"
	//"fmt"

	db "github.com/theduke/dukedb"	
)

type Resource struct {
	Debug bool
	Backend db.Backend
	Hooks ApiHooks
	UserHandler ApiUserHandler

	Model db.Model

	FindOneRequiresAuth bool
	FindRequiresAuth bool
	
	ApiCreateAllowed bool	
	CreateRequiresAuth bool

	ApiDeleteAllowed bool
	DeleteRequiresAuth bool

	ApiUpdateAllowed bool
	UpdateRequiresAuth bool
}

func NewResource(model db.Model, hooks ApiHooks) ApiResource {
	r := Resource{
		ApiCreateAllowed: true,
		ApiUpdateAllowed: true,
		ApiDeleteAllowed: true,
	}
	r.SetModel(model)
	r.SetHooks(hooks)
	return &r
}

func(res *Resource) GetBackend() db.Backend {
	return res.Backend
}

func(res *Resource) SetBackend(x db.Backend) {
	res.Backend = x
}

func(res *Resource) GetDebug() bool {
	return res.Debug
}

func(res *Resource) SetDebug(x bool) {
	res.Debug = x
}

func(res *Resource) GetUserHandler() ApiUserHandler {
	return res.UserHandler
}

func(res *Resource) SetUserHandler(x ApiUserHandler) {
	res.UserHandler = x
}

func(res *Resource) GetModel() db.Model {
	return res.Model
}

func(res *Resource) SetModel(x db.Model) {
	res.Model = x
}

func (res *Resource) SetHooks(h ApiHooks) {
	res.Hooks = h

	if h == nil {
		return
	}

	r := reflect.ValueOf(h)
	if field := r.FieldByName("Debug"); field.IsValid() {
		res.Debug = field.Bool()
	}

	// Set permission fields.
	if field := r.FieldByName("FindOneRequiresAuth"); field.IsValid() {
		res.FindOneRequiresAuth = field.Bool()
	}
	if field := r.FieldByName("FindRequiresAuth"); field.IsValid() {
		res.FindRequiresAuth = field.Bool()
	}

	if field := r.FieldByName("ApiCreateAllowed"); field.IsValid() {
		res.ApiCreateAllowed = field.Bool()
	}
	if field := r.FieldByName("CreateRequiresAuth"); field.IsValid() {
		res.CreateRequiresAuth = field.Bool()
	}

	if field := r.FieldByName("ApiUpdateAllowed"); field.IsValid() {
		res.ApiUpdateAllowed = field.Bool()
	}
	if field := r.FieldByName("UpdateRequiresAuth"); field.IsValid() {
		res.UpdateRequiresAuth = field.Bool()
	}

	if field := r.FieldByName("ApiDeleteAllowed"); field.IsValid() {
		res.ApiDeleteAllowed = field.Bool()
	}
	if field := r.FieldByName("DeleteRequiresAuth"); field.IsValid() {
		res.DeleteRequiresAuth = field.Bool()
	}
}

/**
 * Queries.
 */

func (res Resource) Query(q *db.Query) ([]db.Model, ApiError) {
	return res.Backend.Query(q)
}

func (res Resource) GetQuery() *db.Query {
	return res.Backend.Q(res.Model.GetCollection())
}

/**
 * FindOne
 */

func (res *Resource) FindOne(rawId string) (db.Model, ApiError) {
	return res.Backend.FindOne(res.Model.GetCollection(), rawId)
}

func (res *Resource) ApiFindOne(rawId string, r ApiRequest) ApiResponse {
	findOneHook, ok := res.Hooks.(ApiFindOneHook)
	if ok {
		return findOneHook.ApiFindOne(res, rawId, r)
	}

	user := r.GetUser()
	if res.FindOneRequiresAuth {
		if user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

  result, err := res.FindOne(rawId)
  if err != nil {
  	return Response{Error: err}
  }

  userCanFindOneHook, ok := res.Hooks.(UserCanFindOneHook)
  if ok && !userCanFindOneHook.UserCanFindOne(res, result, user){
  	return NewErrorResponse("permission_denied", "")
  }

  return Response{
  	Data: result,
  }
}


/**
 * Find.
 */

func (res Resource) Find(query *db.Query) ([]db.Model, ApiError) {
	return res.Backend.Query(query)
}

func (res *Resource) ApiFind(query *db.Query, r ApiRequest) ApiResponse {
	user := r.GetUser()

	apiFindHook, ok := res.Hooks.(ApiFindHook)
	if ok {
		return apiFindHook.ApiFind(res, query, r)
	}

	if res.FindRequiresAuth {
		if user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

  result, err := res.Query(query)
  if err != nil {
  	return Response{Error: err}
  }

  canFindHook, ok := res.Hooks.(UserCanFindHook)
  if ok && !canFindHook.UserCanFind(res, result, user) {
  	return NewErrorResponse("permission_denied", "")
  }

  return Response{
  	Data: result,
  }
}


/**
 * Create.
 */

func (res *Resource) Create(obj db.Model, user ApiUser) ApiError {
	if beforeCreate, ok := res.Hooks.(BeforeCreateHook); ok {
		if err := beforeCreate.BeforeCreate(res, obj, user); err != nil {
			return err
		}
	}

	if user != nil {
		if canCreateHook, ok := res.Hooks.(UserCanCreateHook); ok {
			if !canCreateHook.UserCanCreate(res, obj, user) {
				return Error{Code: "permission_denied"}
			}
		}
	}

	if err := res.Backend.Create(obj); err != nil {
		return err
	}

	if afterCreate, ok := res.Hooks.(AfterCreateHook); ok {
		if err := afterCreate.AfterCreate(res, obj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) ApiCreate(obj db.Model, r ApiRequest) ApiResponse {
	if createHook, ok := res.Hooks.(ApiCreateHook); ok {
		return createHook.ApiCreate(res, obj, r)
	}

	if !res.ApiCreateAllowed {
		return NewErrorResponse("not_allowed", "")
	}

	user := r.GetUser()
	if res.CreateRequiresAuth {
		if user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	err := res.Create(obj, user)
	if err != nil {
		return Response{Error: err}
	}

	return Response{
  	Data: obj,
  }
}

/**
 * Update.
 */

func (res *Resource) Update(obj db.Model, user ApiUser) ApiError {
	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return err
	} else if oldObj == nil {
		return Error{Code: "not_found"}
	}

	if beforeUpdate, ok := res.Hooks.(BeforeUpdateHook); ok {
		if err := beforeUpdate.BeforeUpdate(res, obj, oldObj, user); err != nil {
			return err
		}
	}

	if user != nil {
		if canUpdateHook, ok := res.Hooks.(UserCanUpdateHook); ok {
			if !canUpdateHook.UserCanUpdate(res, obj, oldObj, user) {
				return Error{Code: "permission_denied"}
			}
		}
	}

	if err := res.Backend.Update(obj); err != nil {
		return err
	}

	if afterUpdate, ok := res.Hooks.(AfterUpdateHook); ok {
		if err := afterUpdate.AfterUpdate(res, obj, oldObj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) ApiUpdate(obj db.Model, r ApiRequest) ApiResponse {
	if updateHook, ok := res.Hooks.(ApiUpdateHook); ok {
		return updateHook.ApiUpdate(res, obj, r)
	}

	if !res.ApiUpdateAllowed {
		return NewErrorResponse("not_allowed", "")
	}

	user := r.GetUser()
	if res.UpdateRequiresAuth && user == nil {
		return NewErrorResponse("permission_denied", "")
	}

	if err := res.Update(obj, user); err != nil {
		return Response{Error: err}
	}

	return Response{
  	Data: obj,
  }
}

/**
 * Delete.
 */


func (res *Resource) Delete(obj db.Model, user ApiUser) ApiError {
	if beforeDelete, ok := res.Hooks.(BeforeDeleteHook); ok {
		if err := beforeDelete.BeforeDelete(res, obj, user); err != nil {
			return err
		}
	}

	if user != nil {
		if canDeleteHook, ok := res.Hooks.(UserCanDeleteHook); ok {
			if !canDeleteHook.UserCanDelete(res, obj, user) {
				return Error{Code: "permission_denied"}
			}
		}
	}

	if err := res.Backend.Delete(obj); err != nil {
		return err
	}

	if afterDelete, ok := res.Hooks.(AfterDeleteHook); ok {
		if err := afterDelete.AfterDelete(res, obj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) ApiDelete(id string, r ApiRequest) ApiResponse {
	if !res.ApiDeleteAllowed {
		return NewErrorResponse("not_allowed", "")
	}

	user := r.GetUser()
	if res.DeleteRequiresAuth {
		if user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	oldObj, err := res.FindOne(id)
	if err != nil {
		return Response{Error: err}
	} else if oldObj == nil {
		return NewErrorResponse("not_found", "")
	}

	if deleteHook, ok := res.Hooks.(ApiDeleteHook); ok {
		return deleteHook.ApiDelete(res, oldObj, r)
	}

	if err := res.Delete(oldObj, user); err != nil {
		return Response{Error: err}
	}

	return Response{
  	Data: oldObj,
  }
}
