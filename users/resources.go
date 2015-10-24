package users

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app/methods"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/utils"
)

// randomToken creates a random alphanumeric string with a length of 32.
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

// ApiFindOne verifies the session and returns user and profile in meta if valid.
func (SessionResourceHooks) ApiFindOne(res kit.Resource, rawId string, r kit.Request) kit.Response {
	if rawId == "" {
		return kit.NewErrorResponse("empty_token", "Empty token")
	}

	userService := res.Registry().UserService()

	user, session, err := userService.VerifySession(rawId)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	meta := make(map[string]interface{})

	if user != nil {
		userData, err := res.Backend().ModelToMap(user, true, false)
		if err != nil {
			return &kit.AppResponse{Error: apperror.New("marshal_error", err)}
		}
		meta["user"] = userData

		if user.GetProfile() != nil {
			profileData, err := res.Backend().ModelToMap(user.GetProfile(), true, false)
			if err != nil {
				return &kit.AppResponse{Error: apperror.New("marshal_error", err)}
			}
			meta["profile"] = profileData
		}
	}

	return &kit.AppResponse{
		Data: session,
		Meta: meta,
	}
}

// Creating a session is equivalent to logging in.
func (hooks SessionResourceHooks) ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response {
	userService := res.Registry().UserService()

	meta := r.GetMeta()

	isAnonymous, _ := meta.Bool("anonymous")

	// Find user.
	userIdentifier := meta.String("user")
	adaptor := meta.String("adaptor")
	data, _ := meta.Map("auth-data")

	var user kit.User
	if !isAnonymous {
		if adaptor == "" {
			return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.", true)
		}

		if data == nil {
			kit.NewErrorResponse("no_or_invalid_auth_data", "Expected 'auth-data' dictionary in metadata.")
		}

		var err apperror.Error
		user, err = userService.AuthenticateUser(userIdentifier, adaptor, data)
		if err != nil {
			return kit.NewErrorResponse(err)
		}
	}

	session, err := userService.StartSession(user, r.GetFrontend())
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	responseMeta := make(map[string]interface{})

	if !isAnonymous {
		userData, err := res.Backend().ModelToMap(user, true, false)
		if err != nil {
			return &kit.AppResponse{Error: apperror.New("marshal_error", err)}
		}
		responseMeta["user"] = userData

		if user.GetProfile() != nil {
			profileData, err := res.Backend().ModelToMap(user.GetProfile(), true, false)
			if err != nil {
				return &kit.AppResponse{Error: apperror.New("marshal_error", true, err)}
			}
			responseMeta["profile"] = profileData
		}
	}

	return &kit.AppResponse{
		Data: session,
		Meta: responseMeta,
	}
}

func (SessionResourceHooks) ApiDelete(res kit.Resource, id string, r kit.Request) kit.Response {
	if id != r.GetSession().GetStrID() {
		return kit.NewErrorResponse("permission_denied", "Permission denied", 403)
	}

	if err := res.Backend().Delete(r.GetSession()); err != nil {
		return &kit.AppResponse{Error: apperror.Wrap(err, "db_delete_error", true)}
	}

	return &kit.AppResponse{}
}

/**
 * User resource.
 */

type UserResourceHooks struct {
}

func (UserResourceHooks) Methods(res kit.Resource) []kit.Method {
	sendConfirmationEmail := &methods.Method{
		Name:     "users.send-confirmation-email",
		Blocking: false,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
			user := r.GetUser()
			if user == nil {
				return kit.NewErrorResponse("not_authenticated", "")
			}

			if user.IsEmailConfirmed() {
				return kit.NewErrorResponse("email_already_confirmed", "The users email address is already confirmed")
			}

			err := registry.UserService().SendConfirmationEmail(user)
			if err != nil {
				return kit.NewErrorResponse("confirm_failed", "Could not confirm email")
			}

			return &kit.AppResponse{
				Data: map[string]interface{}{"success": true},
			}
		},
	}

	confirmEmail := &methods.Method{
		Name:     "users.confirm-email",
		Blocking: false,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
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

			_, err := registry.UserService().ConfirmEmail(token)
			if err != nil {
				return kit.NewErrorResponse("confirm_failed", "Could not confirm email")
			}

			return &kit.AppResponse{
				Data: map[string]interface{}{"success": true},
			}
		},
	}

	requestPwReset := &methods.Method{
		Name:     "users.request-password-reset",
		Blocking: false,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
			data, ok := r.GetData().(map[string]interface{})
			if !ok {
				return kit.NewErrorResponse("invalid_data", "Expected data dict with 'user' key", true)
			}

			userIdentifier, ok := data["user"].(string)
			if !ok {
				return kit.NewErrorResponse("invalid_data", "Expected data dict with 'user' string key", true)
			}

			rawUser, err := res.Q().Filter("email", userIdentifier).Or("username", userIdentifier).First()
			if err != nil {
				return &kit.AppResponse{Error: err}
			}
			if rawUser == nil {
				return kit.NewErrorResponse("unknown_user", fmt.Sprintf("The user %v does not exist", userIdentifier), true)
			}

			user := rawUser.(kit.User)

			err = registry.UserService().SendPasswordResetEmail(user)
			if err != nil {
				registry.Logger().Errorf("Could not send password reset email for user %v: %v", user, err)
				return kit.NewErrorResponse("reset_email_send_failed", "Could not send the reset password mail.", true)
			}

			return &kit.AppResponse{
				Data: map[string]interface{}{"success": true},
			}
		},
	}

	pwReset := &methods.Method{
		Name:     "users.password-reset",
		Blocking: false,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
			// Verify that token is in data.
			data, ok := r.GetData().(map[string]interface{})
			if !ok {
				return kit.NewErrorResponse("invalid_data", "Expected 'token' key in data", true)
			}
			token, ok := data["token"].(string)
			if !ok {
				return kit.NewErrorResponse("invalid_data", "Expected 'token' string key in data", true)
			}
			if token == "" {
				return kit.NewErrorResponse("empty_token", "", true)
			}

			// Verify that password is in data.
			newPw, ok := data["password"].(string)
			if !ok {
				return kit.NewErrorResponse("invalid_passord", "Expected 'password' string key in data", true)
			}
			if newPw == "" {
				return kit.NewErrorResponse("empty_password", "Password may not be empty", true)
			}

			user, err := registry.UserService().ResetPassword(token, newPw)
			if err != nil {
				if err.IsPublic() {
					return &kit.AppResponse{Error: err}
				} else {
					return kit.NewErrorResponse("password_reset_failed", "Could not reset the password.", true)
				}
			}

			return &kit.AppResponse{
				Data: map[string]interface{}{
					"success":   true,
					"userId":    user.GetID(),
					"userEmail": user.GetEmail(),
				},
			}
		},
	}

	changePassword := &methods.Method{
		Name:     "users.change-password",
		Blocking: false,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
			// Get userId and password from request.
			userId := utils.GetMapStringKey(r.GetData(), "userId")
			if userId == "" {
				return kit.NewErrorResponse("no_userid", "Expected userID key in data", true)
			}
			password := utils.GetMapStringKey(r.GetData(), "password")
			if password == "" {
				return kit.NewErrorResponse("no_password", "Expected password key in data", true)
			}

			// Permission check.
			user := r.GetUser()
			if user == nil {
				return kit.NewErrorResponse("permission_denied", true)
			}

			// Users can only change their own password, unless they are admins.
			if userId != user.GetStrID() {
				if !(user.HasRole("admin") || user.HasPermission("users.change_passwords")) {
					return kit.NewErrorResponse("permission_denied", true)
				}
			}

			// User has the right permissions.
			userService := registry.UserService()

			// Find the user.
			rawUser, err := userService.UserResource().FindOne(userId)
			if err != nil {
				return kit.NewErrorResponse("db_error", true, err)
			}
			if rawUser == nil {
				return kit.NewErrorResponse("user_does_not_exist", true)
			}

			targetUser := rawUser.(kit.User)

			if err := userService.ChangePassword(targetUser, password); err != nil {
				return &kit.AppResponse{Error: err}
			}

			// Everything worked fine.
			return &kit.AppResponse{
				Data: map[string]interface{}{"success": true},
			}
		},
	}

	return []kit.Method{
		AuthenticateMethod,
		sendConfirmationEmail,
		confirmEmail,
		requestPwReset,
		pwReset, changePassword,
	}
}

func (hooks UserResourceHooks) ApiCreate(res kit.Resource, obj kit.Model, r kit.Request) kit.Response {
	meta := r.GetMeta()

	adaptor := meta.String("adaptor")
	if adaptor == "" {
		return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.", true)
	}

	rawData, ok := meta.Get("auth-data")
	if !ok {
		return kit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.", true)
	}

	data, ok := rawData.(map[string]interface{})
	if !ok {
		return kit.NewErrorResponse("invalid_auth_data", "Invalid auth data: expected dictionary", true)
	}

	user := obj.(kit.User)

	service := res.Registry().UserService()

	// If a profile model was registered, and profile data is in meta,
	// create the profile model.
	if profiles := service.ProfileResource(); profiles != nil {
		profile := profiles.CreateModel().(kit.UserProfile)

		if rawData, ok := meta.Get("profile"); ok {
			if data, ok := rawData.(map[string]interface{}); ok {
				// Profile data present in meta.
				// Update profile with data.
				info := res.Backend().ModelInfo(profile.Collection())
				if err := db.UpdateModelFromData(info, profile, data); err != nil {
					return &kit.AppResponse{
						Error: apperror.Wrap(err, "invalid_profile_data", "Invalid profile data.", true),
					}
				}
			}
		}

		user.SetProfile(profile)
	}

	if err := service.CreateUser(user, adaptor, data); err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: user,
	}
}

func (hooks UserResourceHooks) AllowFind(res kit.Resource, obj kit.Model, user kit.User) bool {
	/*
		u := obj.(kit.User)
		return u.GetID() == user.GetID()
	*/
	return true
}

func (hooks UserResourceHooks) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if user.HasRole("admin") || user.HasPermission("users.update") {
		return true
	}
	return obj.GetID() == user.GetID()
}

func (hooks UserResourceHooks) AllowDelete(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	return false
}

/**
 * Roles resource.
 */

type RoleResourceHooks struct {
	resources.AdminResource
}

// Restrict querying to admins.
func (RoleResourceHooks) AllowFind(res kit.Resource, model kit.Model, user kit.User) bool {
	return user != nil && user.HasRole("admin")
}

func (RoleResourceHooks) ApiAlterQuery(res kit.Resource, query db.Query, r kit.Request) apperror.Error {
	// Join permissions.
	query.Join("Permissions")
	return nil
}

type PermissionResourceHooks struct {
	resources.AdminResource
}
