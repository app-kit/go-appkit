package users

import (
	"time"

	"github.com/theduke/go-appkit/users/auth"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/error"
	"github.com/theduke/go-appkit/resources"
)

type Service struct {
	Users     kit.Resource
	Sessions  kit.Resource
	AuthItems kit.Resource

	Roles       kit.Resource
	Permissions kit.Resource

	profileModel kit.UserProfile

	AuthAdaptors map[string]kit.AuthAdaptor
}

// Ensure UserService implements kit.UserService.
var _ kit.UserService = (*Service)(nil)

func NewService(profileModel kit.UserProfile) *Service {
	h := Service{
		profileModel: profileModel,
	}

	h.AuthAdaptors = make(map[string]kit.AuthAdaptor)

	// Register auth adaptors.
	h.AddAuthAdaptor(auth.AuthAdaptorPassword{})

	// Build resources.
	users := resources.NewResource(&BaseUserIntID{}, UserResourceHooks{
		ProfileModel: profileModel,
	})
	h.Users = users

	sessions := resources.NewResource(&BaseSessionIntID{}, SessionResourceHooks{})
	h.Sessions = sessions

	auths := resources.NewResource(&BaseAuthItemIntID{}, nil)
	h.AuthItems = auths

	roles := resources.NewResource(&Role{}, RoleResourceHooks{})
	h.Roles = roles

	permissions := resources.NewResource(&Permission{}, PermissionResourceHooks{})
	h.Permissions = permissions

	return &h
}

func (h *Service) AuthAdaptor(name string) kit.AuthAdaptor {
	return h.AuthAdaptors[name]
}

func (h *Service) AddAuthAdaptor(a kit.AuthAdaptor) {
	h.AuthAdaptors[a.GetName()] = a
}

func (h *Service) UserResource() kit.Resource {
	return h.Users
}

func (h *Service) SetUserResource(x kit.Resource) {
	h.Users = x
}

func (h *Service) SessionResource() kit.Resource {
	return h.Sessions
}

func (h *Service) SetSessionResource(x kit.Resource) {
	h.Sessions = x
}

func (h *Service) AuthItemResource() kit.Resource {
	return h.AuthItems
}

func (h *Service) SetAuthItemResource(x kit.Resource) {
	h.AuthItems = x
}

func (h *Service) ProfileModel() kit.UserProfile {
	return h.profileModel
}

/**
 * RBAC resources.
 */

func (u *Service) RoleResource() kit.Resource {
	return u.Roles
}

func (u *Service) SetRoleResource(x kit.Resource) {
	u.Roles = x
}

func (u *Service) PermissionResource() kit.Resource {
	return u.Permissions
}

func (u *Service) SetPermissionResource(x kit.Resource) {
	u.Permissions = x
}

func (h *Service) CreateUser(user kit.User, adaptorName string, authData interface{}) Error {
	adaptor := h.AuthAdaptor(adaptorName)
	if adaptor == nil {
		return AppError{Code: "unknown_auth_adaptor"}
	}

	data, err := adaptor.BuildData(user, authData)
	if err != nil {
		return AppError{Code: "adaptor_error", Message: err.Error()}
	}

	if user.GetUsername() == "" {
		user.SetUsername(user.GetEmail())
	}

	// Check if user with same username or email exists.
	oldUser, err2 := h.Users.Q().
		Filter("email", user.GetEmail()).Or("username", user.GetUsername()).First()
	if err2 != nil {
		return err2
	} else if oldUser != nil {
		return AppError{
			Code:    "user_exists",
			Message: "A user with the username or email already exists",
		}
	}

	user.SetIsActive(true)

	if h.profileModel != nil && user.GetProfile() == nil {
		newProfile, _ := h.Users.Backend().NewModel(h.profileModel.Collection())
		user.SetProfile(newProfile.(kit.UserProfile))
	}

	if err := h.Users.Create(user, nil); err != nil {
		return err
	}

	// Create profile if one exists.
	if profile := user.GetProfile(); profile != nil {
		profile.SetID(user.GetID())
		if err := h.Users.Backend().Create(profile); err != nil {
			h.Users.Backend().Delete(user)
			return err
		}
	}

	rawAuth, _ := h.AuthItems.Backend().NewModel(h.AuthItems.Model().Collection())
	auth := rawAuth.(kit.AuthItem)
	auth.SetUserID(user.GetID())
	auth.SetType(adaptorName)
	auth.SetData(data)

	if err := h.AuthItems.Create(auth, nil); err != nil {
		h.Users.Delete(user, nil)
		return AppError{Code: "auth_save_failed", Message: err.Error()}
	}

	return nil
}

func (h *Service) AuthenticateUser(user kit.User, authAdaptorName string, data interface{}) Error {
	if !user.IsActive() {
		return AppError{Code: "user_inactive"}
	}

	authAdaptor := h.AuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return AppError{
			Code:    "unknown_auth_adaptor",
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	rawAuth, err := h.AuthItems.Q().
		Filter("typ", authAdaptorName).And("user_id", user.GetID()).First()

	if err != nil {
		return err
	}

	auth := rawAuth.(kit.AuthItem)

	cleanData, err2 := auth.GetData()
	if err2 != nil {
		return AppError{
			Code:    "invalid_auth_data",
			Message: err.Error(),
		}
	}

	ok, err2 := authAdaptor.Authenticate(user, cleanData, data)
	if err2 != nil {
		return AppError{Code: "auth_error", Message: err.Error()}
	}
	if !ok {
		return AppError{Code: "invalid_credentials"}
	}

	return nil
}

func (h *Service) VerifySession(token string) (kit.User, kit.Session, Error) {
	rawSession, err := h.Sessions.FindOne(token)
	if err != nil {
		return nil, nil, err
	}
	if rawSession == nil {
		return nil, nil, AppError{Code: "session_not_found"}
	}
	session := rawSession.(kit.Session)

	// Load user.
	rawUser, err := h.UserResource().FindOne(session.GetUserID())
	if err != nil {
		return nil, nil, err
	}
	user := rawUser.(kit.User)

	if !user.IsActive() {
		return nil, nil, AppError{Code: "user_inactive"}
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, nil, AppError{Code: "session_expired"}
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := h.Sessions.Update(session, nil); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
