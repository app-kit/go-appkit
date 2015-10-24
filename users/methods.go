package users

import (
	"fmt"
	"reflect"

	//"github.com/theduke/go-apperror"
	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app/methods"
)

var AuthenticateMethod kit.Method = &methods.Method{
	Name:     "users.authenticate",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		fmt.Printf("\nuser: %+v\n", reflect.TypeOf(r.GetUser()))

		if r.GetUser() != nil {
			return kit.NewErrorResponse("already_authenticated", "Can't authenticate a session which is already authenticated", true)
		}

		data, ok := r.GetData().(map[string]interface{})
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Invalid data: expected dict", true)
		}

		// Find user.

		userIdentifier, _ := data["user"].(string)

		adaptor, _ := data["adaptor"].(string)
		if adaptor == "" {
			return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.", true)
		}

		authData, _ := data["auth-data"].(map[string]interface{})
		if authData == nil {
			kit.NewErrorResponse("no_or_invalid_auth_data", "Expected 'auth-data' dictionary in metadata.")
		}

		userService := registry.UserService()
		user, err := userService.AuthenticateUser(userIdentifier, adaptor, authData)
		if err != nil {
			return kit.NewErrorResponse(err)
		}

		session := r.GetSession()

		if session != nil {
			// Already have a session, so update it.
			if err := session.SetUserID(user.GetID()); err != nil {
				return kit.NewErrorResponse(err)
			}
			if err := userService.SessionResource().Update(session, user); err != nil {
				return kit.NewErrorResponse(err)
			}
		} else {
			session, err = userService.StartSession(user, r.GetFrontend())
			if err != nil {
				return kit.NewErrorResponse(err)
			}
		}

		// Set user in session to include it in serialized response.
		// Also required for wamp backend.
		session.SetUser(user)

		return &kit.AppResponse{
			Data: session,
		}
	},
}
