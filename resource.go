package appkit

import (

)

type Resource struct {
	Debug bool
	Backend Backend

	UserHandler ApiUserHandler

	Model ApiModel
	BuildQuery func(rawQuery RawQuery) (Query, ApiError)

	FindOneRequiresAuth bool
	UserCanFindOne func(obj ApiModel, user ApiUser) bool

	FindRequiresAuth bool
	AfterFind func(objs []ApiModel) ApiError
	UserCanFind func(objs []ApiModel, user ApiUser) bool

	CreateRequiresAuth bool
	BeforeCreate func(obj ApiModel) ApiError
	AfterCreate func(obj ApiModel) ApiError
	UserCanCreate func(obj ApiModel, user ApiUser) bool

	DeleteRequiresAuth bool
	BeforeDelete func(obj ApiModel) ApiError
	AfterDelete func(obj ApiModel) ApiError
	UserCanDelete func(obj ApiModel, user ApiUser) bool

	UpdateRequiresAuth bool
	BeforeUpdate func(obj, oldobj ApiModel) ApiError
	AfterUpdate func(obj, oldobj ApiModel) ApiError
	UserCanUpdate func(obj ApiModel, old ApiModel, user ApiUser) bool
}

func(res *Resource) GetBackend() Backend {
	return res.Backend
}

func(res *Resource) SetBackend(x Backend) {
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

func(res *Resource) GetModel() ApiModel {
	return res.Model
}

func(res *Resource) SetModel(x ApiModel) {
	res.Model = x
}

/**
 * FindOne
 */

func (res Resource) FindOne(rawId string) (ApiModel, ApiError) {
	return res.Backend.FindOne(res.Model.GetName(), rawId)
}

func (res Resource) FindOneBy(filters map[string]interface{}) (ApiModel, ApiError) {
	return res.Backend.FindOneBy(res.Model.GetName(), filters)
}

func (res Resource) ApiFindOne(rawId string, r ApiRequest) ApiResponse {
	var user ApiUser	
	if res.FindOneRequiresAuth {
		if user = r.GetUser(); user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

  result, err := res.FindOne(rawId)
  if err != nil {
  	return Response{Error: err}
  }

  if res.UserCanFindOne != nil && !res.UserCanFindOne(result, user) {
  	return NewErrorResponse("permission_denied", "")
  }

  return Response{
  	Data: result,
  }
}


/**
 * Find.
 */

func (res Resource) FindBy(filters map[string]interface{}) ([]ApiModel, ApiError) {
	return res.Backend.FindBy(res.Model.GetName(), filters)
}

func (res Resource) Find(query Query) ([]ApiModel, ApiError) {
	result, err := res.Backend.Find(query)
  if err != nil {
  	return nil, err
  }

  if res.AfterFind != nil {
	  if err := res.AfterFind(result); err != nil {
			return nil, err
		}
  }

  return result, nil
}

func (res Resource) ApiFind(rawQuery RawQuery, r ApiRequest) ApiResponse {
	var user ApiUser	
	if res.FindRequiresAuth {
		if user = r.GetUser(); user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	query, err := res.BuildQuery(rawQuery)
	if err != nil {
		return Response{Error: err}
	}

  result, err := res.Find(query)
  if err != nil {
  	return Response{Error: err}
  }

  if res.UserCanFind != nil && !res.UserCanFind(result, user) {
  	return NewErrorResponse("permission_denied", "")
  }

  return Response{
  	Data: result,
  }
}


/**
 * Create.
 */

func (res Resource) Create(obj ApiModel) ApiError {
	if res.BeforeCreate != nil {
		if err := res.BeforeCreate(obj); err != nil {
			return err
		}
	}

	if err := res.Backend.Create(obj); err != nil {
		return err
	}

	if res.AfterCreate != nil {
		if err := res.AfterCreate(obj); err != nil {
			return err
		}
	}

	return nil
}

func (res Resource) ApiCreate(obj ApiModel, r ApiRequest) ApiResponse {
	var user ApiUser	
	if res.CreateRequiresAuth {
		if user = r.GetUser(); user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	if res.UserCanCreate != nil && !res.UserCanCreate(obj, user) {
  	return NewErrorResponse("permission_denied", "")
  }

	err := res.Create(obj)
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

func (res Resource) Update(obj ApiModel) ApiError {
	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return err
	}

	if oldObj == nil {
		return Error{Code: "not_found"}
	}

	if res.BeforeUpdate != nil {
		if err := res.BeforeUpdate(obj, oldObj); err != nil {
			return err
		}
	}

	if err := res.Backend.Update(obj); err != nil {
		return err
	}

	if res.AfterUpdate != nil {
		if err := res.AfterUpdate(obj, oldObj); err != nil {
			return err
		}
	}

	return nil
}

func (res Resource) ApiUpdate(obj ApiModel, r ApiRequest) ApiResponse {
	var user ApiUser	
	if res.UpdateRequiresAuth {
		if user = r.GetUser(); user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return Response{
			Error: err,
		}
	}

	if oldObj == nil {
		return NewErrorResponse("record_not_found", "")
	}

	if res.BeforeUpdate != nil {
		if err := res.BeforeUpdate(obj, oldObj); err != nil {
			return Response{Error: err}
		}
	}

	if res.UserCanUpdate != nil && !res.UserCanUpdate(obj, oldObj, user) {
  	return NewErrorResponse("permission_denied", "")
  }

	if err := res.Backend.Update(obj); err != nil {
		return NewErrorResponse("db_error", err.Error())
	}

	if res.AfterUpdate != nil {
		if err := res.AfterUpdate(obj, oldObj); err != nil {
			return Response{Error: err}
		}
	}

	return Response{
  	Data: obj,
  }
}

/**
 * Delete.
 */


func (res Resource) Delete(obj ApiModel) ApiError {
	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return err
	}

	if oldObj == nil {
		return Error{Code: "not_found"}
	}

	if res.BeforeDelete != nil {
		if err := res.BeforeDelete(obj); err != nil {
			return err
		}
	}

	if err := res.Backend.Delete(obj); err != nil {
		return err
	}

	if res.AfterDelete != nil {
		if err := res.AfterDelete(obj); err != nil {
			return err
		}
	}

	return nil
}

func (res Resource) ApiDelete(id string, r ApiRequest) ApiResponse {
	var user ApiUser	
	if res.DeleteRequiresAuth {
		if user = r.GetUser(); user == nil {
			return NewErrorResponse("permission_denied", "")
		}
	}

	oldObj, err := res.FindOne(id)
	if err != nil {
		return Response{
			Error: err,
		}
	}
	if oldObj == nil {
		return NewErrorResponse("record_not_found", "")
	}

	if res.BeforeDelete != nil {
		if err := res.BeforeDelete(oldObj); err != nil {
			return Response{Error: err}
		}
	}

	if res.UserCanDelete != nil && !res.UserCanDelete(oldObj, user) {
  	return NewErrorResponse("permission_denied", "")
  }

	if err := res.Backend.Delete(oldObj); err != nil {
		return NewErrorResponse("db_error", err.Error())
	}

	if res.AfterDelete != nil {
		if err := res.AfterDelete(oldObj); err != nil {
			return Response{Error: err}
		}
	}

	return Response{
  	Data: oldObj,
  }
}
