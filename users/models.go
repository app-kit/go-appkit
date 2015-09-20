package users

import (
	"encoding/json"
	"strconv"
	"time"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type BaseAuthItem struct {
	UserID string `sql:"-" db:"primary-key"`
	Typ    string `sql:"size: 100; not null"`

	Data string `sql:type:text; not null`
}

func (a *BaseAuthItem) Collection() string {
	return "auth_items"
}

func (a *BaseAuthItem) GetName() string {
	return "auth_items"
}

func (a BaseAuthItem) TableName() string {
	return "auth_items"
}

func (a *BaseAuthItem) GetID() string {
	return ""
}

func (a *BaseAuthItem) SetID(x string) error {
	return nil
}

func (b *BaseAuthItem) SetUserID(rawId string) {
	b.UserID = rawId
}

func (b *BaseAuthItem) GetUserID() string {
	return b.UserID
}

func (b *BaseAuthItem) SetType(x string) {
	b.Typ = x
}

func (b *BaseAuthItem) GetType() string {
	return b.Typ
}

func (b *BaseAuthItem) SetData(data interface{}) kit.Error {
	json, err := json.Marshal(data)
	if err != nil {
		return kit.WrapError(err, "auth_item_json_marshal_error", "")
	}

	b.Data = string(json)
	return nil
}

func (b *BaseAuthItem) GetData() (interface{}, kit.Error) {
	var data interface{}
	if err := json.Unmarshal([]byte(b.Data), &data); err != nil {
		return nil, kit.WrapError(err, "auth_item_json_unmarshal_error", "")
	}

	return data, nil
}

type BaseAuthItemIntID struct {
	BaseAuthItem
	ID     uint64
	UserID uint64 `gorm:"primary-key" sql:"not null;"`
}

func (u *BaseAuthItemIntID) SetUserID(x string) {
	i, err := strconv.ParseUint(x, 10, 64)
	if err != nil {
		return
	}

	u.UserID = i
}

func (u *BaseAuthItemIntID) GetUserID() string {
	return strconv.FormatUint(u.UserID, 10)
}

type BaseUser struct {
	Active bool `sql:"not null"`

	Username string `db:"unique;not-null"`
	Email    string `db:"unique;not-null"`

	EmailConfirmed bool

	LastLogin time.Time `jsonapi:"name=last-login"`

	Data string `db:"ignore-zero;max:10000"`

	CreatedAt time.Time `jsonapi:"name=created-at"`
	UpdatedAt time.Time `jsonapi:"name=updated-at"`

	Roles []*Role `gorm:"many2many:user_roles;" db:"m2m;"`
}

func (u BaseUser) Collection() string {
	return "users"
}

// For api2go!
func (a *BaseUser) GetName() string {
	return "users"
}

func (a BaseUser) TableName() string {
	return "users"
}

func (a BaseUser) GetProfile() kit.UserProfile {
	return nil
}

func (a BaseUser) SetProfile(p kit.UserProfile) {

}

// Implement User interface.

func (u *BaseUser) SetIsActive(x bool) {
	u.Active = x
}

func (u *BaseUser) IsActive() bool {
	return u.Active
}

func (u *BaseUser) SetEmail(x string) {
	u.Email = x
}

func (u *BaseUser) GetEmail() string {
	return u.Email
}

func (u *BaseUser) IsEmailConfirmed() bool {
	return u.EmailConfirmed
}

func (u *BaseUser) SetIsEmailConfirmed(x bool) {
	u.EmailConfirmed = x
}

func (u *BaseUser) SetUsername(x string) {
	u.Username = x
}

func (u *BaseUser) GetUsername() string {
	return u.Username
}

func (u *BaseUser) SetLastLogin(x time.Time) {
	u.LastLogin = x
}

func (u *BaseUser) GetLastLogin() time.Time {
	return u.LastLogin
}

func (u *BaseUser) GetData() (interface{}, kit.Error) {
	if u.Data == "" {
		return nil, nil
	}
	var data interface{}
	err := json.Unmarshal([]byte(u.Data), &data)
	if err != nil {
		return nil, kit.AppError{
			Code:     "json_marshal_error",
			Message:  err.Error(),
			Internal: true,
		}
	}
	return data, nil
}

func (u *BaseUser) SetData(x interface{}) kit.Error {
	js, err := json.Marshal(x)
	if err != nil {
		return kit.AppError{
			Code:     "json_marshal_error",
			Message:  err.Error(),
			Internal: true,
		}
	}
	u.Data = string(js)
	return nil
}

func (u *BaseUser) SetCreatedAt(x time.Time) {
	u.CreatedAt = x
}

func (u *BaseUser) GetCreatedAt() time.Time {
	return u.CreatedAt
}

func (u *BaseUser) SetUpdatedAt(x time.Time) {
	u.UpdatedAt = x
}

func (u *BaseUser) GetUpdatedAt() time.Time {
	return u.UpdatedAt
}

/**
 * RBAC methods.
 */

func (u *BaseUser) GetRoles() []kit.Role {
	slice := make([]kit.Role, 0)
	for _, r := range u.Roles {
		slice = append(slice, r)
	}
	return slice
}

func (u *BaseUser) AddRole(r kit.Role) {
	if u.Roles == nil {
		u.Roles = make([]*Role, 0)
	}
	if !u.HasRole(r) {
		u.Roles = append(u.Roles, r.(*Role))
	}
}

func (u *BaseUser) RemoveRole(r kit.Role) {
	if u.Roles == nil {
		return
	}

	for i := 0; i < len(u.Roles); i++ {
		if u.Roles[i].Name == r.GetName() {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
		}
	}
}

func (u *BaseUser) ClearRoles() {
	u.Roles = make([]*Role, 0)
}

func (u *BaseUser) HasRole(r kit.Role) bool {
	return u.HasRoleStr(r.GetName())
}

func (u *BaseUser) HasRoleStr(role string) bool {
	if u.Roles == nil {
		return false
	}

	for i := 0; i < len(u.Roles); i++ {
		if u.Roles[i].Name == role {
			return true
		}
	}

	return false
}

type BaseUserStrID struct {
	BaseUser
	ID string
}

func (u *BaseUserStrID) SetID(x string) error {
	u.ID = x
	return nil
}

func (u *BaseUserStrID) GetID() string {
	return u.ID
}

type BaseUserIntID struct {
	BaseUser

	ID uint64 `gorm:"primary-key" sql:"not null"`
}

// For api2go!
func (u BaseUserIntID) GetName() string {
	return "users"
}

func (u *BaseUserIntID) SetID(x string) error {
	i, err := strconv.ParseUint(x, 10, 64)
	if err != nil {
		return err
	}

	u.ID = i
	return nil
}

func (u *BaseUserIntID) GetID() string {
	return strconv.FormatUint(u.ID, 10)
}

/**
 * Base UserProfile.
 */

type BaseUserProfile struct {
	UserID string `sql:"-"`
}

// TODO: fix.
func (p *BaseUserProfile) SetUserID(x string) {
	p.UserID = x
}

func (p BaseUserProfile) GetUserID() string {
	return p.UserID
}

type BaseUserProfileIntID struct {
	UserID uint64 `gorm:"primary-key" sql:"not null"`
}

func (p *BaseUserProfileIntID) SetUserID(x string) {
	id, _ := strconv.ParseUint(x, 10, 64)
	p.UserID = id
}

func (p *BaseUserProfileIntID) GetUserID() string {
	return strconv.FormatUint(p.UserID, 10)
}

type Token struct {
	UserModelIntID

	Type      string `db:"notnull"`
	Token     string `db:"primary-key;unique;notnull;omit-zero"`
	ExpiresAt time.Time
}

// Ensure that Token implements Token interface.
var _ kit.UserToken = (*Token)(nil)

func (t *Token) Collection() string {
	return "user_tokens"
}

func (t *Token) GetID() string {
	return t.Token
}

func (t *Token) SetID(x string) error {
	t.Token = x
	return nil
}

func (t *Token) GetType() string {
	return t.Type
}

func (t *Token) SetType(x string) {
	t.Type = x
}

func (t *Token) GetToken() string {
	return t.Token
}

func (t *Token) SetToken(x string) {
	t.Token = x
}

func (t *Token) GetExpiresAt() time.Time {
	return t.ExpiresAt
}

func (t *Token) SetExpiresAt(tm time.Time) {
	t.ExpiresAt = tm
}

func (t *Token) IsValid() bool {
	return t.ExpiresAt.IsZero() || t.ExpiresAt.Sub(time.Now()) > 0
}

/**
 * BaseSession
 */

type BaseSession struct {
	Token      string    `gorm:"primary-key" db:"primary-key" sql:"size:100"`
	UserID     string    `sql:"-"`
	StartedAt  time.Time `sql:"not null" jsonapi:"name=started-at"`
	ValidUntil time.Time `sql:"not null" jsonapi:"name=valid-until"`

	Typ string `sql:"size:100; not null"`
}

func (b BaseSession) Collection() string {
	return "sessions"
}

// For api2go.
func (b BaseSession) GetName() string {
	return "sessions"
}

func (b BaseSession) TableName() string {
	return "sessions"
}

func (s BaseSession) GetID() string {
	return s.Token
}

func (s *BaseSession) SetID(x string) error {
	s.Token = x
	return nil
}

func (s *BaseSession) GetType() string {
	return s.Typ
}

func (s *BaseSession) SetType(x string) {
	s.Typ = x
}

func (s *BaseSession) SetToken(x string) {
	s.Token = x
}

func (s *BaseSession) GetToken() string {
	return s.Token
}

func (s *BaseSession) SetUserID(x string) {
	s.UserID = x
}

func (s *BaseSession) GetUserID() string {
	return s.UserID
}

func (s *BaseSession) SetStartedAt(x time.Time) {
	s.StartedAt = x
}

func (s *BaseSession) GetStartedAt() time.Time {
	return s.StartedAt
}

func (s *BaseSession) SetValidUntil(x time.Time) {
	s.ValidUntil = x
}

func (s *BaseSession) GetValidUntil() time.Time {
	return s.ValidUntil
}

func (s *BaseSession) IsGuest() bool {
	return s.UserID == ""
}

type BaseSessionIntID struct {
	BaseSession
	UserID uint64 `gorm:"primary-key" sql:"not null;"`
}

func (u *BaseSessionIntID) SetUserID(x string) {
	i, _ := strconv.ParseUint(x, 10, 64)
	u.UserID = i
}

func (u *BaseSessionIntID) GetUserID() string {
	return strconv.FormatUint(u.UserID, 10)
}

func (s *BaseSessionIntID) IsGuest() bool {
	return s.UserID == 0
}

func (s *BaseSessionIntID) MarshalJSON() ([]byte, error) {
	return db.ModelToJson(nil, s)
}

/**
 * Role.
 */

type Role struct {
	Name        string        `gorm:"primary-key" db:"primary-key" sql:"type: varchar(200)"`
	Permissions []*Permission `gorm:"many2many:role_permissions;" db:"m2m"`
}

func (r Role) Collection() string {
	return "roles"
}

func (r Role) GetTableName() string {
	return "roles"
}

func (r *Role) SetName(n string) {
	r.Name = n
}

func (r Role) GetName() string {
	return r.Name
}

func (r Role) GetID() string {
	return r.Name
}

func (r *Role) SetID(n string) error {
	r.Name = n
	return nil
}

func (r *Role) GetPermissions() []kit.Permission {
	perms := make([]kit.Permission, 0)
	for _, p := range r.Permissions {
		perms = append(perms, p)
	}
	return perms
}

/**
 * Permission.
 */

type Permission struct {
	Name string `gorm:"primary-key" db:"primary-key" sql:"type: varchar(200)"`
}

func (r Permission) Collection() string {
	return "permissions"
}

func (r Permission) GetTableName() string {
	return "permissions"
}

func (r *Permission) SetName(n string) {
	r.Name = n
}

func (r Permission) GetName() string {
	return r.Name
}

func (p Permission) GetID() string {
	return p.Name
}

func (p *Permission) SetID(n string) error {
	p.Name = n
	return nil
}

/**
 * Extendable models that are related to a user.
 */

type UserModelStrID struct {
	db.BaseModelStrID

	User   *BaseUserStrID
	UserID string
}

func (m *UserModelStrID) GetUser() kit.User {
	return m.User
}

func (m *UserModelStrID) SetUser(x kit.User) {
	m.User = x.(*BaseUserStrID)
	m.SetUserID(x.GetID())
}

func (m *UserModelStrID) GetUserID() string {
	return m.UserID
}

func (m *UserModelStrID) SetUserID(x string) error {
	m.UserID = x
	return nil
}

type UserModelIntID struct {
	User   *BaseUserIntID
	UserID uint64
}

func (m *UserModelIntID) GetUser() kit.User {
	return m.User
}

func (m *UserModelIntID) SetUser(x kit.User) {
	m.User = x.(*BaseUserIntID)
	m.SetUserID(x.GetID())
}

func (m *UserModelIntID) GetUserID() string {
	return strconv.FormatUint(m.UserID, 10)
}

func (m *UserModelIntID) SetUserID(rawId string) error {
	id, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		return err
	}
	m.UserID = id
	return nil
}
