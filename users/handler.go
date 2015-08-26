package users

import(
	"time"

	kit "github.com/theduke/appkit"
	"github.com/theduke/appkit/users/auth"
)

type UserHandler struct {
	Users kit.ApiResource
	Sessions kit.ApiResource
	AuthItems kit.ApiResource

	AuthAdaptors map[string]kit.ApiAuthAdaptor
}

func NewUserHandler() *UserHandler {
	h := UserHandler{}
	h.AuthAdaptors = make(map[string]kit.ApiAuthAdaptor)

	// Register auth adaptors.
	h.AddAuthAdaptor(auth.AuthAdaptorPassword{})

	// Build resources.
	users := kit.NewResource(&BaseUserIntID{}, UserResourceHooks{})
	h.Users = users

	sessions := kit.NewResource(&BaseSessionIntID{}, SessionResourceHooks{
		ApiUpdateAllowed: false,
		ApiDeleteAllowed: false,
	})
	h.Sessions = sessions

	auths := kit.NewResource(&BaseAuthItemIntID{}, nil)
	h.AuthItems = auths

	return &h
}

func (h *UserHandler) GetAuthAdaptor(name string) kit.ApiAuthAdaptor {
	return h.AuthAdaptors[name];
}

func (h *UserHandler) AddAuthAdaptor(a kit.ApiAuthAdaptor) {
	h.AuthAdaptors[a.GetName()] = a
}

func(h *UserHandler) GetUserResource() kit.ApiResource {
	return h.Users
}

func(h *UserHandler) SetUserResource(x kit.ApiResource) {
	h.Users = x
}

func(h *UserHandler) GetSessionResource() kit.ApiResource {
	return h.Sessions
}

func(h *UserHandler) SetSessionResource(x kit.ApiResource) {
	h.Sessions = x
}

func(h *UserHandler) GetAuthItemResource() kit.ApiResource {
	return h.AuthItems
}

func(h *UserHandler) SetAuthItemResource(x kit.ApiResource) {
	h.AuthItems = x
}

func (h *UserHandler) CreateUser(user kit.ApiUser, adaptorName string, authData interface{}) kit.ApiError {
	adaptor := h.GetAuthAdaptor(adaptorName)
	if adaptor == nil  {
		return kit.Error{Code: "unknown_auth_adaptor"}
	}

	data, err := adaptor.BuildData(authData)
	if err != nil {
		return kit.Error{Code: "adaptor_error", Message: err.Error()}
	}

	if user.GetUsername() == "" {
		user.SetUsername(user.GetEmail())
	}
	
	// Check if user with same username or email exists.
	oldUser, err2 := h.Users.GetQuery().
	  Filter("email", user.GetEmail()).Or("username", user.GetUsername()).First()
	if err2 != nil {
		return err2
	} else if oldUser != nil {
		return kit.Error{
			Code: "user_exists", 
			Message: "A user with the username or email already exists",
		}
	}

	if err := h.Users.Create(user, nil); err != nil {
		return err
	}

	rawAuth, _ := h.AuthItems.GetBackend().NewModel(h.AuthItems.GetModel().GetCollection())
	auth := rawAuth.(kit.ApiAuthItem)
	auth.SetUserID(user.GetID())
	auth.SetType(adaptorName)
	auth.SetData(data)

	if err := h.AuthItems.Create(auth, nil); err != nil {
		h.Users.Delete(user, nil)
		return kit.Error{Code: "auth_save_failed", Message: err.Error()}
	}

	return nil
}

func (h *UserHandler) AuthenticateUser(user kit.ApiUser, authAdaptorName string, data interface{}) kit.ApiError {
	if !user.IsActive() {
		return kit.Error{Code: "user_inactive"}
	}

	authAdaptor := h.GetAuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return kit.Error{
			Code: "unknown_auth_adaptor", 
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	rawAuth, err := h.AuthItems.GetQuery().
	  Filter("typ", authAdaptorName).And("user_id", user.GetID()).First()

	if err != nil {
		return err
	}

	auth := rawAuth.(kit.ApiAuthItem)

	cleanData, err2 := auth.GetData()
	if err2 != nil {
		return kit.Error{
			Code: "invalid_auth_data", 
			Message: err.Error(),
		}
	}

	ok, err2 := authAdaptor.Authenticate(cleanData, data)
	if err2 != nil {
		return kit.Error{Code: "auth_error", Message: err.Error()}
	}
	if !ok {
		return kit.Error{Code: "invalid_credentials"}
	}

	return nil
}

func (h *UserHandler) VerifySession(token string) (kit.ApiUser, kit.ApiSession, kit.ApiError) {
	rawSession, err := h.Sessions.FindOne(token)
	if err != nil {
		return nil, nil, err
	}
	if rawSession == nil {
		return nil, nil, kit.Error{Code: "session_not_found"}
	}
	session := rawSession.(kit.ApiSession)

	// Load user.
	rawUser, err := h.GetUserResource().FindOne(session.GetUserID())
	if err != nil {
		return nil, nil, err
	}
	user := rawUser.(kit.ApiUser)

	if !user.IsActive() {
		return nil, nil, kit.Error{Code: "user_inactive"}
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, nil, kit.Error{Code: "session_expired"}
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := h.Sessions.Update(session, nil); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
