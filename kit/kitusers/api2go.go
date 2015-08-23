package kitusers

import (
	//"log"
	"reflect"
	"errors"

	"github.com/manyminds/api2go"
	"github.com/theduke/appkit/users"
	"github.com/theduke/appkit/kit"
)

type UserResource struct {
	kit.GormResource
}

func (r *UserResource) Setup() {
	r.FindOneRequiresAuth = true
	r.FindAllRequiresAuth = true
	r.CreateRequiresAuth = true
	r.UpdateRequiresAuth = true
	r.DeleteRequiresAuth = true

	r.UserCanFindAll = func(objs []interface{}, user users.User) bool {
		return false
	}

	r.UserCanFindOne = func(obj interface{}, curUser users.User) bool {
		// Allow user to only read his own data.
		user, _ := obj.(users.User)
		return user.GetID() == curUser.GetID()
	}
}

func (res UserResource) Create(result interface{}, r api2go.Request) (api2go.Responder, error) {
	user, ok := result.(users.User)
	if !ok {
		return nil, api2go.NewHTTPError(errors.New("invalid_user_instance"), "invalid user instance", 500)
	}

	// Get auth data.
	authField := reflect.ValueOf(result).Elem().FieldByName("AuthData")
	if !authField.IsValid() {
		return nil, api2go.NewHTTPError(errors.New("no_auth_data"), "User object does not have authdata", 500)
	}
	authData := authField.Interface()


	interfaceUser, _ := result.(users.User)
	err := res.UserHandler.CreateUser(interfaceUser, "password", authData)
	if err != nil {
		return nil, api2go.NewHTTPError(err, err.Error(), 500)
	}

	return kit.ApiResponse{Status: 201, Res: user}, nil
}
