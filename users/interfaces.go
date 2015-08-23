package users

import (
	"time"
)

type AuthAdaptor interface {
	GetName() string

	BuildData(interface{}) (interface{}, error)
	Authenticate(interface{}, interface{}) (bool, error)
}

type AuthItem interface {
	SetUserID(string)
	GetUserID() string

	SetType(string)
	GetType() string

	SetData(interface{}) error
	GetData() (interface{}, error)
}


type AuthProvider interface {
	NewItem(string, string, interface{}) AuthItem

	Create(AuthItem) error
	Update(AuthItem) error
	Delete(AuthItem) error

	FindOne(string, User) (AuthItem, error)
	FindByUser(User) ([]AuthItem, error)
}

type UserProfile interface {
	SetUserID(string)
	GetUserID() string
}

type User interface {
	SetID(string) error
	GetID() string

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

type UserProvider interface {
	NewUser() User

	Create(User) error
	Update(User) error
	Delete(User) error

	FindOne(string) (User, error)
	FindByUsername(string) (User, error)
	FindByEmail(string) (User, error)

	FindAll() ([]User, error)
}

type Session interface {
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

type SessionProvider interface {
	NewSession() Session

	Create(Session) error
	Update(Session) error
	Delete(Session) error

	FindOne(string) (Session, error)
	FindAll() ([]Session, error)
}

type EventHandler interface {
	OnBeforeSignup(User) error
	OnAfterSignup(User)
}
