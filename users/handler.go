package users

import(
	"time"

	"github.com/theduke/appkit"
	"github.com/theduke/appkit/users/auth"
)

type UserHandler struct {
	Users appkit.ApiResource
	Sessions appkit.ApiResource
	AuthItems appkit.ApiResource

	AuthAdaptors map[string]appkit.ApiAuthAdaptor
}

func NewUserHandler() *UserHandler {
	h := UserHandler{}
	h.AuthAdaptors = make(map[string]appkit.ApiAuthAdaptor)

	// Register auth adaptors.
	h.AddAuthAdaptor(auth.AuthAdaptorPassword{})

	// Build resources.
	users := UserResource{}
	users.Model = &BaseUserIntID{}
	h.Users = &users

	sessions := SessionResource{}
	sessions.Model = &BaseSessionIntID{}
	h.Sessions = &sessions

	auths := appkit.Resource{}
	auths.Model = &BaseAuthItemIntID{}
	h.AuthItems = &auths

	return &h
}

func (h *UserHandler) GetAuthAdaptor(name string) appkit.ApiAuthAdaptor {
	return h.AuthAdaptors[name];
}

func (h *UserHandler) AddAuthAdaptor(a appkit.ApiAuthAdaptor) {
	h.AuthAdaptors[a.GetName()] = a
}

func(h *UserHandler) GetUserResource() appkit.ApiResource {
	return h.Users
}

func(h *UserHandler) SetUserResource(x appkit.ApiResource) {
	h.Users = x
}

func(h *UserHandler) GetSessionResource() appkit.ApiResource {
	return h.Sessions
}

func(h *UserHandler) SetSessionResource(x appkit.ApiResource) {
	h.Sessions = x
}

func(h *UserHandler) GetAuthItemResource() appkit.ApiResource {
	return h.AuthItems
}

func(h *UserHandler) SetAuthItemResource(x appkit.ApiResource) {
	h.AuthItems = x
}

func (h *UserHandler) CreateUser(user appkit.ApiUser, adaptorName string, authData interface{}) appkit.ApiError {
	adaptor := h.GetAuthAdaptor(adaptorName)
	if adaptor == nil  {
		return appkit.Error{Code: "unknown_auth_adaptor"}
	}

	data, err := adaptor.BuildData(authData)
	if err != nil {
		return appkit.Error{Code: "adaptor_error", Message: err.Error()}
	}

	if user.GetUsername() == "" {
		user.SetUsername(user.GetEmail())
	}
	
	// Check if user with same username or email exists.
	q := map[string]interface{}{"email": user.GetEmail()}
	if u, err := h.Users.FindOneBy(q); err != nil {
		return err
	} else if u != nil {
		return appkit.Error{Code: "email_exists"}
	}

	q = map[string]interface{}{"username": user.GetUsername()}
	if u, err := h.Users.FindOneBy(q); err != nil {
		return err
	} else if u != nil {
		return appkit.Error{Code: "username_exists"}
	}

	if err := h.Users.GetBackend().Create(user); err != nil {
		return err
	}

	rawAuth, _ := h.AuthItems.GetBackend().GetType(h.AuthItems.GetModel().GetName())
	auth := rawAuth.(appkit.ApiAuthItem)
	auth.SetUserID(user.GetID())
	auth.SetType(adaptorName)
	auth.SetData(data)

	if err := h.AuthItems.Create(auth); err != nil {
		h.Users.Delete(user)
		return appkit.Error{Code: "auth_save_failed", Message: err.Error()}
	}

	return nil
}

func (h *UserHandler) AuthenticateUser(user appkit.ApiUser, authAdaptorName string, data interface{}) appkit.ApiError {
	if !user.IsActive() {
		return appkit.Error{Code: "user_inactive"}
	}

	authAdaptor := h.GetAuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return appkit.Error{
			Code: "unknown_auth_adaptor", 
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	rawAuth, err := h.AuthItems.FindOneBy(map[string]interface{}{
		"typ": authAdaptorName,
		"user_id": user.GetID(),
	})
	if err != nil {
		return appkit.Error{Code: "auth_error", Message: err.Error()}
	}

	auth := rawAuth.(appkit.ApiAuthItem)

	cleanData, err2 := auth.GetData()
	if err2 != nil {
		return appkit.Error{
			Code: "invalid_auth_data", 
			Message: err.Error(),
		}
	}

	ok, err2 := authAdaptor.Authenticate(cleanData, data)
	if err2 != nil {
		return appkit.Error{Code: "auth_error", Message: err.Error()}
	}
	if !ok {
		return appkit.Error{Code: "invalid_credentials"}
	}

	return nil
}

func (h *UserHandler) VerifySession(token string) (appkit.ApiUser, appkit.ApiSession, appkit.ApiError) {
	rawSession, err := h.Sessions.FindOneBy(map[string]interface{}{"token": token})
	if err != nil {
		return nil, nil, err
	}
	if rawSession == nil {
		return nil, nil, appkit.Error{Code: "session_not_found"}
	}
	session := rawSession.(appkit.ApiSession)

	// Load user.
	rawUser, err := h.GetUserResource().FindOne(session.GetUserID())
	if err != nil {
		return nil, nil, err
	}
	user := rawUser.(appkit.ApiUser)

	if !user.IsActive() {
		return nil, nil, appkit.Error{Code: "user_inactive"}
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, nil, appkit.Error{Code: "session_expired"}
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := h.Sessions.Update(session); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
