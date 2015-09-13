package appkit

import (
	db "github.com/theduke/go-dukedb"
	"time"
)

type ApiAuthAdaptor interface {
	GetName() string

	BuildData(interface{}) (interface{}, error)
	Authenticate(interface{}, interface{}) (bool, error)
}

type ApiAuthItem interface {
	db.Model

	SetUserID(string)
	GetUserID() string

	SetType(string)
	GetType() string

	SetData(interface{}) error
	GetData() (interface{}, error)
}

/**
 * Users
 */

type ApiUserHandler interface {
	CreateUser(user ApiUser, adaptor string, data interface{}) ApiError
	AuthenticateUser(user ApiUser, adaptor string, data interface{}) ApiError
	VerifySession(token string) (ApiUser, ApiSession, ApiError)

	GetAuthAdaptor(name string) ApiAuthAdaptor
	AddAuthAdaptor(a ApiAuthAdaptor)

	SetUserResource(ApiResource)
	GetUserResource() ApiResource

	GetProfileModel() ApiUserProfile

	SetSessionResource(ApiResource)
	GetSessionResource() ApiResource

	SetAuthItemResource(ApiResource)
	GetAuthItemResource() ApiResource

	SetRoleResource(ApiResource)
	GetRoleResource() ApiResource

	SetPermissionResource(ApiResource)
	GetPermissionResource() ApiResource
}

type ApiUserProfile interface {
	db.Model
}

type ApiUser interface {
	db.Model

	SetIsActive(bool)
	IsActive() bool

	SetUsername(string)
	GetUsername() string

	SetEmail(string)
	GetEmail() string

	SetLastLogin(time.Time)
	GetLastLogin() time.Time

	SetCreatedAt(time.Time)
	GetCreatedAt() time.Time

	SetUpdatedAt(time.Time)
	GetUpdatedAt() time.Time

	SetProfile(ApiUserProfile)
	GetProfile() ApiUserProfile

	GetRoles() []ApiRole
	AddRole(ApiRole)
	RemoveRole(ApiRole)
	ClearRoles()
	HasRole(ApiRole) bool
	HasRoleStr(string) bool
}

type ApiUserModel interface {
	db.Model
	User() ApiUser
	SetUser(ApiUser)

	UserID() string
	SetUserID(string) error
}

type ApiSession interface {
	db.Model

	SetType(string)
	GetType() string

	SetToken(string)
	GetToken() string

	SetUserID(string)
	GetUserID() string

	SetStartedAt(time.Time)
	GetStartedAt() time.Time

	SetValidUntil(time.Time)
	GetValidUntil() time.Time

	IsGuest() bool
}

type ApiRole interface {
	GetName() string
	SetName(string)
}

type ApiPermission interface {
	GetName() string
	SetName(string)
}
