package kit

import (
	//"strconv"
	"errors"
	"net/http"
	"reflect"
	//"log"
	"strconv"
	//"reflect"
	//"encoding/json"

	"github.com/manyminds/api2go"
	"github.com/jinzhu/gorm"

	"github.com/theduke/appkit/users"
)

type ApiResponse struct {
	Res  interface{}
	Status int
	Meta map[string]interface{}
}

func (r ApiResponse) Metadata() map[string]interface{} {
	return r.Meta
}

func (r ApiResponse) Result() interface{} {
	return r.Res
}

func (r ApiResponse) StatusCode() int {
	return r.Status
}

type GormResource struct {
	Debug bool
	Db *gorm.DB

	UserHandler *users.BaseUserHandler
	ItemType interface{}

	FindOneRequiresAuth bool
	FindAllRequiresAuth bool
	CreateRequiresAuth bool
	UpdateRequiresAuth bool
	DeleteRequiresAuth bool

	BeforeFindOne func(query *gorm.DB, r api2go.Request) *ApiResponse
	FindOneQuery func(id string, query *gorm.DB, r api2go.Request) *gorm.DB
	AfterFindOne func(obj interface{}, r api2go.Request) *ApiResponse
	UserCanFindOne func(obj interface{}, user users.User) bool

	FindAllQuery func(query *gorm.DB, r api2go.Request) *gorm.DB
	BeforeFindAll func(query *gorm.DB, r api2go.Request) *ApiResponse
	AfterFindAll func(objs []interface{}, r api2go.Request) *ApiResponse
	UserCanFindAll func(objs []interface{}, user users.User) bool

	BeforeCreate func(obj interface{}, r api2go.Request) *ApiResponse
	AfterCreate func(obj interface{}, r api2go.Request) *ApiResponse
	UserCanCreate func(obj interface{}, user users.User) bool

	BeforeDelete func(obj interface{}, r api2go.Request) *ApiResponse
	AfterDelete func(obj interface{}, r api2go.Request) *ApiResponse
	UserCanDelete func(obj interface{}, user users.User) bool

	BeforeUpdate func(obj, oldObj interface{}, r api2go.Request) *ApiResponse
	AfterUpdate func(obj, oldObj interface{}, r api2go.Request) *ApiResponse
	UserCanUpdate func(obj interface{}, old interface{}, user users.User) bool
}


func (r *GormResource) Init(itemType interface{}, db *gorm.DB, userHandler *users.BaseUserHandler) {
	r.ItemType = itemType
	r.Db = db
	r.UserHandler = userHandler

	r.Setup()
}

func (r *GormResource) Setup() {

}

func (res *GormResource) OnBeforeHandle(request api2go.Request) api2go.Responder {
	request.Context.Set("session", nil)	
	request.Context.Set("user", nil)


	// Authenticate user if possible.
	token := request.Header.Get("Authentication")
	if token != "" {
		session, err := res.UserHandler.VerifySession(token)
		if err != nil {
			return ApiResponse{
				Status: 500,
				Res: api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500),
			}
		}

		if session != nil {
			user, err := res.UserHandler.Users.FindOne(session.GetUserID())
			if err != nil {
				return ApiResponse{
					Status: 500,
					Res: api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500),
				}
			}

			request.Context.Set("session", session)	
			request.Context.Set("user", user)
		}
	}

	return nil
}

func (res *GormResource) OnAfterHandle(request api2go.Request, response api2go.Responder) {
	
}

func (res *GormResource) GetQuery() *gorm.DB {
	q := res.Db

	if res.Debug {
		q = q.Debug()
	}

	return q
}

/**
 * Find one.
 */

func (res GormResource) FindOne(rawId string, r api2go.Request) (api2go.Responder, error) {
	var user users.User	
	if res.FindOneRequiresAuth {
		if u := r.Context.Get("user"); u == nil {
			return ApiResponse{
				Status: 403,
			}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
		} else {
			user = u.(users.User)
		}
	}

  result := reflect.ValueOf(res.ItemType).Interface()

  query := res.GetQuery()

 	if res.FindOneQuery != nil {
  	query = res.FindOneQuery(rawId, query, r)
 	} else {
 		intId, _ := strconv.Atoi(rawId)
		query = query.Where("id = ?", intId)
	}

  if res.BeforeFindOne != nil {
		if resp := res.BeforeFindOne(query, r); resp != nil {
			return resp, nil
		}
  }

  if err := query.First(&result).Error; err != nil {
  	if err.Error() == "record not found" {
  		return ApiResponse{Status: 404}, api2go.NewHTTPError(errors.New("not_found"), "not_found", 404)
		} else {
			return ApiResponse{Status: 500}, api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
		}
  }

  if res.UserCanFindOne != nil && !res.UserCanFindOne(result, user) {
  	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
  }

  if res.AfterFindOne != nil {
	  if resp := res.AfterFindOne(result, r); resp != nil {
			return resp, nil
		}
  }

  return ApiResponse{Status: 200, Res: result}, nil
}

/**
 * Find All.
 */

func (res *GormResource) FindAll(r api2go.Request) (api2go.Responder, error) {
	var user users.User	
	if res.FindAllRequiresAuth {
		if u := r.Context.Get("user"); u == nil {
			return ApiResponse{
				Status: 403,
			}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
		} else {
			user = u.(users.User)
		}
	}

	var result []interface{}

	query := res.GetQuery()

	if res.FindAllQuery != nil {
		query = res.FindAllQuery(query, r)
	}

	if res.BeforeFindAll != nil {
		if resp := res.BeforeFindAll(query, r); resp != nil {
			return resp, nil
		}
	}

	if err := query.Find(&result).Error; err != nil {
		return ApiResponse{Status: 500}, api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
	}

	if res.UserCanFindAll != nil && !res.UserCanFindAll(result, user) {
  	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
  }

  if res.AfterFindAll != nil {
		if resp := res.AfterFindAll(result, r); resp != nil {
			return resp, nil
		}
  }

	return ApiResponse{Status: 200, Res: result}, nil
}

/**
 * Create.
 */

func (res GormResource) Create(result interface{}, r api2go.Request) (api2go.Responder, error) {
	var user users.User	
	if res.CreateRequiresAuth {
		if u := r.Context.Get("user"); u == nil {
			return ApiResponse{
				Status: 403,
			}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
		} else {
			user = u.(users.User)
		}
	}

	if res.BeforeCreate != nil {
		if resp := res.BeforeCreate(result, r); resp != nil {
			return resp, nil
		}
	}

	if res.UserCanCreate != nil && !res.UserCanCreate(result, user) {
  	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
  }

	if err := res.Db.Create(&result).Error; err != nil {
		return ApiResponse{Status: 500}, api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
	}

	if res.AfterCreate != nil {
		if resp := res.AfterCreate(result, r); resp != nil {
			return resp, nil
		}
	}

	return ApiResponse{Res: result, Status: http.StatusCreated}, nil
}

/**
 * Delete.
 */


func (res GormResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
	var user users.User	
	if res.DeleteRequiresAuth {
		if u := r.Context.Get("user"); u == nil {
			return ApiResponse{
				Status: 403,
			}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
		} else {
			user = u.(users.User)
		}
	}

	id, _ := strconv.Atoi(rawId)

	result := reflect.ValueOf(res.ItemType).Interface()
	if err := res.Db.Where("id = ?", id).First(&result).Error; err != nil {
		return ApiResponse{Status: 500}, api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
	}

	if res.BeforeDelete != nil {
		if resp := res.BeforeDelete(result, r); resp != nil {
			return resp, nil
		}
	}

	if res.UserCanDelete != nil && !res.UserCanDelete(result, user) {
  	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
  }

  if err:= res.Db.Delete(&result).Error; err != nil {
  	return ApiResponse{Status: 500}, api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
  }
 	
 	if res.AfterDelete != nil {
	  if resp := res.AfterDelete(result, r); resp != nil {
			return resp, nil
		}
 	} 

  return ApiResponse{Status: http.StatusNoContent}, nil
}


/**
 * Update.
 */

func (res GormResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
	var user users.User	
	if res.UpdateRequiresAuth {
		if u := r.Context.Get("user"); u == nil {
			return ApiResponse{
				Status: 403,
			}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
		} else {
			user = u.(users.User)
		}
	}

	refl := reflect.ValueOf(res.ItemType)

	field := refl.FieldByName("id")
	if !field.IsValid() || field.Int() < 1 {
		return ApiResponse{Status: 401}, 
		  api2go.NewHTTPError(errors.New("no_id"), "", 401)
	}

	id := field.Int()

	oldResult := refl.Interface()
	if err := res.Db.Where("id = ?", id).First(&oldResult).Error; err != nil {
		return ApiResponse{Status: 500}, 
		  api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
	}

	if res.BeforeUpdate != nil {
		if resp := res.BeforeUpdate(obj, oldResult, r); resp != nil {
			return resp, nil
		}
	}

	if res.UserCanUpdate != nil && !res.UserCanUpdate(obj, oldResult, user) {
  	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403)
  }

	if err := res.Db.Save(&obj).Error; err != nil {
		return ApiResponse{Status: 500}, 
		  api2go.NewHTTPError(errors.New("db_error"), err.Error(), 500)
	}
	
	if res.AfterUpdate != nil {
		if resp := res.AfterUpdate(obj, oldResult, r); resp != nil {
			return resp, nil
		}
	}	

	return ApiResponse{Status: 200, Res: obj}, nil
}

/**
 * Read only
 */

type GormReadOnlyResource struct {
 	GormResource
}

func (res GormReadOnlyResource) Update(obj interface{}, r api2go.Request) (api2go.Responder, error) {
 	return ApiResponse{
			Status: 403,
		}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403) 
}

func (res GormReadOnlyResource) Delete(rawId string, r api2go.Request) (api2go.Responder, error) {
 	return ApiResponse{
		Status: 403,
	}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403) 
}

func (res GormReadOnlyResource) Create(result interface{}, r api2go.Request) (api2go.Responder, error) {
	return ApiResponse{
		Status: 403,
	}, api2go.NewHTTPError(errors.New("permission_denied"), "", 403) 
}