package users

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app/methods"
)

func randomToken() string {
	n := 32

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	symbols := big.NewInt(int64(len(alphanum)))
	states := big.NewInt(0)
	states.Exp(symbols, big.NewInt(int64(n)), nil)
	r, err := rand.Int(rand.Reader, states)
	if err != nil {
		return ""
	}

	var bytes = make([]byte, n)
	r2 := big.NewInt(0)
	symbol := big.NewInt(0)
	for i := range bytes {
		r2.DivMod(r, symbols, symbol)
		r, r2 = r2, r
		bytes[i] = alphanum[symbol.Int64()]
	}
	return string(bytes)
}

type SessionResourceHooks struct {
	UpdateAllowed    bool
	ApiDeleteAllowed bool
}

func StartSession(res kit.Resource, user kit.User) (kit.Session, kit.Error) {
	token := randomToken()
	if token == "" {
		return nil, kit.AppError{Code: "token_creation_failed"}
	}

	rawSession, err := res.Backend().NewModel(res.Model().Collection())
	if err != nil {
		return nil, err
	}
	session := rawSession.(kit.Session)

	session.SetUserID(user.GetID())
	session.SetToken(token)
	session.SetStartedAt(time.Now())
	session.SetValidUntil(time.Now().Add(time.Hour * 12))

	err = res.Create(session, nil)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (hooks SessionResourceHooks) ApiCreate(res kit.Resource, obj db.Model, r kit.Request) kit.Response {
	meta := r.GetMeta()

	userIdentifier := meta.String("user")
	if userIdentifier == "" {
		return kit.NewErrorResponse("user_missing", "Expected 'user' in metadata.")
	}

	// Find user.
	userResource := res.Dependencies().UserService().UserResource()

	rawUser, err := userResource.Q().
		Filter("username", userIdentifier).Or("email", userIdentifier).First()

	if err != nil {
		return &kit.AppResponse{Error: err}
	} else if rawUser == nil {
		return kit.NewErrorResponse("user_not_found", "User not found for identifier: "+userIdentifier)
	}

	user := rawUser.(kit.User)

	adaptor := meta.String("adaptor")
	if adaptor == "" {
		return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		kit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	err = res.Dependencies().UserService().AuthenticateUser(user, adaptor, data)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	session, err := StartSession(res, user)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: session,
	}
}

/**
 * User resource.
 */

type UserResourceHooks struct {
	ProfileModel kit.UserProfile
}

func (UserResourceHooks) Methods(res kit.Resource) []kit.Method {
	deps := res.Dependencies()

	sendConfirmationEmail := methods.NewMethod("users.send-confirmation-email", false, func(a kit.App, r kit.Request, unblock func()) kit.Response {
		user := r.GetUser()
		if user == nil {
			return kit.NewErrorResponse("not_authenticated", "")
		}

		err := deps.UserService().SendConfirmationEmail(user)
		if err != nil {
			return kit.NewErrorResponse("confirm_failed", "Could not confirm email")
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"success": true},
		}
	})

	confirmEmail := methods.NewMethod("users.confirm-email", false, func(a kit.App, r kit.Request, unblock func()) kit.Response {
		deps.Logger().Infof("data: %+v\n", r.GetData())
		data, ok := r.GetData().(map[string]interface{})
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected data dict with 'token' key")
		}
		token, ok := data["token"].(string)
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected 'token' string key in data")
		}
		if token == "" {
			return kit.NewErrorResponse("empty_token", "")
		}

		_, err := deps.UserService().ConfirmEmail(token)
		if err != nil {
			return kit.NewErrorResponse("confirm_failed", "Could not confirm email")
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"success": true},
		}
	})

	requestPwReset := methods.NewMethod("users.request-password-reset", false, func(a kit.App, r kit.Request, unblock func()) kit.Response {
		deps := res.Dependencies()

		data, ok := r.GetData().(map[string]interface{})
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected data dict with 'user' key")
		}

		userIdentifier, ok := data["user"].(string)
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected data dict with 'user' string key")
		}

		rawUser, err := res.Q().Filter("email", userIdentifier).Or("username", userIdentifier).First()
		if err != nil {
			return &kit.AppResponse{Error: err}
		}
		if rawUser == nil {
			return kit.NewErrorResponse("unknown_user", fmt.Sprintf("The user %v does not exist", userIdentifier))
		}

		user := rawUser.(kit.User)

		err = deps.UserService().SendPasswordResetEmail(user)
		if err != nil {
			deps.Logger().Errorf("Could not send password reset email for user %v: %v", user, err)
			return kit.NewErrorResponse("reset_email_send_failed", "Could not send the reset password mail.")
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{"success": true},
		}
	})

	pwReset := methods.NewMethod("users.password-reset", false, func(a kit.App, r kit.Request, unblock func()) kit.Response {
		// Verify that token is in data.
		data, ok := r.GetData().(map[string]interface{})
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected 'token' key in data")
		}
		token, ok := data["token"].(string)
		if !ok {
			return kit.NewErrorResponse("invalid_data", "Expected 'token' string key in data")
		}
		if token == "" {
			return kit.NewErrorResponse("empty_token", "")
		}

		// Verify that password is in data.
		newPw, ok := data["password"].(string)
		if !ok {
			return kit.NewErrorResponse("invalid_passord", "Expected 'password' string key in data")
		}
		if newPw == "" {
			return kit.NewErrorResponse("empty_password", "Password may not be empty")
		}

		deps := res.Dependencies()

		user, err := deps.UserService().ResetPassword(token, newPw)
		if err != nil {
			return kit.NewErrorResponse("password_reset_failed", "Could not reset the password.")
		}

		return &kit.AppResponse{
			Data: map[string]interface{}{
				"success":   true,
				"userId":    user.GetID(),
				"userEmail": user.GetEmail(),
			},
		}
	})

	return []kit.Method{sendConfirmationEmail, confirmEmail, requestPwReset, pwReset}
}

func (hooks UserResourceHooks) ApiCreate(res kit.Resource, obj db.Model, r kit.Request) kit.Response {
	meta := r.GetMeta()

	adaptor := meta.String("adaptor")
	if adaptor == "" {
		return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		return kit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	user := obj.(kit.User)
	if err := res.Dependencies().UserService().CreateUser(user, adaptor, data); err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: user,
	}
}

func (hooks UserResourceHooks) AllowFind(res kit.Resource, obj db.Model, user kit.User) bool {
	u := obj.(kit.User)
	return u.GetID() == user.GetID()
}

func (hooks UserResourceHooks) AllowUpdate(res kit.Resource, obj db.Model, old db.Model, user kit.User) bool {
	return user != nil && obj.GetID() == user.GetID()
}

func (hooks UserResourceHooks) AllowDelete(res kit.Resource, obj db.Model, old db.Model, user kit.User) bool {
	return false
}

type RoleResourceHooks struct {
}

type PermissionResourceHooks struct {
}
