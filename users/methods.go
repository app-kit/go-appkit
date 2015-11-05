package users

import (
	//"github.com/theduke/go-apperror"
	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/app/methods"
)

var AuthenticateMethod kit.Method = &methods.Method{
	Name:     "users.authenticate",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
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

		authData, _ := data["authData"].(map[string]interface{})
		if authData == nil {
			kit.NewErrorResponse("no_or_invalid_auth_data", "Expected 'authData' dictionary in metadata.")
		}

		userService := registry.UserService()
		user, err := userService.AuthenticateUser(userIdentifier, adaptor, authData)
		if err != nil {
			return kit.NewErrorResponse(err)
		}

		session := r.GetSession()

		// Already have a session, so update it.
		if err := session.SetUserId(user.GetId()); err != nil {
			return kit.NewErrorResponse(err)
		}
		if err := userService.SessionResource().Update(session, user); err != nil {
			return kit.NewErrorResponse(err)
		}

		// Update session with user to include it in response, and also to
		// update the session in case it is persistent (eg in wamp frontend).
		session.SetUser(user)

		return &kit.AppResponse{
			Data: session,
		}
	},
}

var ResumeSessionMethod kit.Method = &methods.Method{
	Name:     "users.resume_session",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		registry.Logger().Infof("data: %v", r.GetData())
		data, _ := r.GetData().(map[string]interface{})
		token, _ := data["token"].(string)

		if token == "" {
			return kit.NewErrorResponse("no_token", "Expected 'token' in data.", true)
		}

		user, session, err := registry.UserService().VerifySession(token)
		if err != nil {
			return kit.NewErrorResponse(err)
		}

		// Update session with user to include it in response, and also to
		// update the session in case it is persistent (eg in wamp frontend).
		session.SetUser(user)

		curSession := r.GetSession()
		curSession.SetToken(token)
		curSession.SetValidUntil(session.GetValidUntil())
		curSession.SetUser(user)

		return &kit.AppResponse{
			Data: session,
		}
	},
}

var UnAuthenticateMethod kit.Method = &methods.Method{
	Name:     "users.unauthenticate",
	Blocking: true,
	Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
		if r.GetUser() == nil {
			return kit.NewErrorResponse("not_authenticated", "Can't un-authenticate a session which is not authenticated", true)
		}

		session := r.GetSession()
		session.SetUser(nil)
		if err := registry.UserService().SessionResource().Update(session, r.GetUser()); err != nil {
			return kit.NewErrorResponse(err)
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{},
		}
	},
}
