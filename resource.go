package appkit

import (
	db "github.com/theduke/dukedb"	
)

type Resource struct {
	Debug bool
	Backend db.Backend
	Hooks ApiHooks
	UserHandler ApiUserHandler

	Model db.Model
}

func NewResource(model db.Model, hooks ApiHooks) ApiResource {
	r := Resource{}
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

func (res *Resource) NewModel() db.Model {
	 n, err := res.Backend.NewModel(res.Model.GetCollection())
	 if err != nil {
	 	return nil
	 }
	 return n.(db.Model)
}

func (res *Resource) SetHooks(h ApiHooks) {
	res.Hooks = h
}

/**
 * Queries.
 */

/**
 * Perform a query.
 */
func (res Resource) Query(q *db.Query) ([]db.Model, ApiError) {
	return res.Backend.Query(q)
}

/**
 * Return a new query initialized with the backend.
 */
func (res Resource) Q() *db.Query {
	return res.Backend.Q(res.Model.GetCollection())
}

/**
 * FindOne
 */

func (res *Resource) FindOne(rawId string) (db.Model, ApiError) {
	return res.Backend.FindOne(res.Model.GetCollection(), rawId)
}

/**
 * Find.
 */

func (res Resource) Find(query *db.Query) ([]db.Model, ApiError) {
	return res.Backend.Query(query)
}

func (res *Resource) ApiFindOne(rawId string, r ApiRequest) ApiResponse {
  result, err := res.FindOne(rawId)
  if err != nil {
    return &Response{Error: err}
  } else if result == nil {
  	return NewErrorResponse("not_found", "")
  }

  user := r.GetUser()
  if allowFind, ok := res.Hooks.(AllowFindHook); ok {
	  if !allowFind.AllowFind(res, result, user) {
	  	return NewErrorResponse("permission_denied", "")
	  }
  }

  return &Response{
  	Data: result,
  }
}

func (res *Resource) ApiFind(query *db.Query, r ApiRequest) ApiResponse {
	apiFindHook, ok := res.Hooks.(ApiFindHook)
	if ok {
		return apiFindHook.ApiFind(res, query, r)
	}

	if alterQuery, ok := res.Hooks.(ApiAlterQueryHook); ok {
		alterQuery.ApiAlterQuery(res, query, r)
	}

  result, err := res.Query(query)
  if err != nil {
  	return &Response{Error: err}
  }

	user := r.GetUser()
  if allowFind, ok := res.Hooks.(AllowFindHook); ok {
  	for _, item := range result {
		  if !allowFind.AllowFind(res, item, user) {
		  	return NewErrorResponse("permission_denied", "")
		  }
  	}
  }

  return &Response{
  	Data: result,
  }
}

func (res *Resource) ApiFindPaginated(query *db.Query, r ApiRequest) ApiResponse {
	resp := res.ApiFind(query, r)
	if resp.GetError() == nil {
		count, _ := res.Backend.Count(query)
		resp.SetMeta(map[string]interface{}{"count": count})
	}

	return resp
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

	user := r.GetUser()
	if allowCreate, ok := res.Hooks.(AllowCreateHook); ok {
		if !allowCreate.AllowCreate(res, obj, user) {
			return NewErrorResponse("permission_denied", "")
		}
	}

	err := res.Create(obj, user)
	if err != nil {
		return &Response{Error: err}
	}

	return &Response{
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

	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return &Response{Error: err}
	} else if oldObj == nil {
		return NewErrorResponse("not_found", "")
	}

	user := r.GetUser()
	if allowUpdate, ok := res.Hooks.(AllowUpdateHook); ok {
		if !allowUpdate.AllowUpdate(res, obj, oldObj, user) {
			return NewErrorResponse("permission_denied", "")
		}
	}

	if beforeUpdate, ok := res.Hooks.(BeforeUpdateHook); ok {
		if err := beforeUpdate.BeforeUpdate(res, obj, oldObj, user); err != nil {
			return &Response{Error: err}
		}
	}

	if err := res.Backend.Update(obj); err != nil {
		return &Response{Error: err}
	}

	if afterUpdate, ok := res.Hooks.(AfterUpdateHook); ok {
		if err := afterUpdate.AfterUpdate(res, obj, oldObj, user); err != nil {
			return &Response{Error: err}
		}
	}

	return &Response{
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
	oldObj, err := res.FindOne(id)
	if err != nil {
		return &Response{Error: err}
	} else if oldObj == nil {
		return NewErrorResponse("not_found", "")
	}

	user := r.GetUser()
	if allowDelete, ok := res.Hooks.(AllowDeleteHook); ok {
		if !allowDelete.AllowDelete(res, oldObj, user) {
			return NewErrorResponse("permission_denied", "")
		}
	}

	if deleteHook, ok := res.Hooks.(ApiDeleteHook); ok {
		return deleteHook.ApiDelete(res, oldObj, r)
	}

	if err := res.Delete(oldObj, user); err != nil {
		return &Response{Error: err}
	}

	return &Response{
  	Data: oldObj,
  }
}

/**
 * Read only resource hooks template
 */

type ReadOnlyResource struct {}

func (r ReadOnlyResource) AllowCreate(res ApiResource, obj db.Model, user ApiUser) bool {
	return false
}

func (r ReadOnlyResource) AllowUpdate(res ApiResource, obj db.Model, user ApiUser) bool {
	return false
}

func (r ReadOnlyResource) AllowDelete(res ApiResource, obj db.Model, user ApiUser) bool {
	return false
}
