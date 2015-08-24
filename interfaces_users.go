package appkit

import (
	"time"
)

type ApiAuthAdaptor interface {
	GetName() string

	BuildData(interface{}) (interface{}, error)
	Authenticate(interface{}, interface{}) (bool, error)
}

type ApiAuthItem interface {
	ApiModel

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

type ApiUserHandler interface{
	CreateUser(user ApiUser, adaptor string, data interface{}) ApiError
	AuthenticateUser(user ApiUser, adaptor string, data interface{}) ApiError
	VerifySession(token string) (ApiUser, ApiSession, ApiError)

	GetAuthAdaptor(name string) ApiAuthAdaptor
	AddAuthAdaptor(a ApiAuthAdaptor)

	SetUserResource(ApiResource)
	GetUserResource() ApiResource
	SetSessionResource(ApiResource)
	GetSessionResource() ApiResource
	SetAuthItemResource(ApiResource)
	GetAuthItemResource() ApiResource
}

type ApiUserProfile interface {
	SetUserID(string)
	GetUserID() string
}

type ApiUser interface {
	ApiModel

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
}

type ApiSession interface {
	ApiModel

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
