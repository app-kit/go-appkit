package users

import (
	"encoding/json"
	"strconv"
	"time"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

/**
 * Extendable models that are related to a user.
 */

type UserModel struct {
	User   *User
	UserID string
}

func (m *UserModel) GetUserID() string {
	return m.UserID
}

func (m *UserModel) SetUserID(x string) error {
	m.UserID = x
	return nil
}

func (m *UserModel) GetUser() kit.User {
	return m.User
}

func (m *UserModel) SetUser(u kit.User) {
	m.User = u.(*User)
	m.SetUserID(u.GetID())
}

type IntUserModel struct {
	User   *IntUser
	UserID uint64
}

func (m *IntUserModel) GetUserID() string {
	if m.UserID == 0 {
		return ""
	}
	return strconv.FormatUint(m.UserID, 10)
}

func (m *IntUserModel) SetUserID(rawId string) error {
	if rawId == "" {
		m.UserID = 0
		return nil
	}

	id, err := strconv.ParseUint(rawId, 10, 64)
	if err != nil {
		return err
	}
	m.UserID = id
	return nil
}

func (m *IntUserModel) GetUser() kit.User {
	return m.User
}

func (m *IntUserModel) SetUser(x kit.User) {
	m.User = x.(*IntUser)
	m.SetUserID(x.GetID())
}

/**
 * User.
 */

type User struct {
	db.BaseModel

	Active bool

	Username string `db:"unique;not-null"`
	Email    string `db:"unique;not-null"`

	EmailConfirmed bool

	LastLogin time.Time `db:"ignore-zero"`

	Data string `db:"ignore-zero;max:10000"`

	CreatedAt time.Time `db:"ignore-zero"`
	UpdatedAt time.Time `db:"ignore-zero"`

	Roles []*Role `db:"m2m;"`
}

// Ensure User implements kit.User interface.
var _ kit.User = (*User)(nil)

func (u User) Collection() string {
	return "users"
}

// Implement User interface.

func (a User) GetProfile() kit.UserProfile {
	return nil
}

func (a User) SetProfile(p kit.UserProfile) {

}

func (u *User) SetIsActive(x bool) {
	u.Active = x
}

func (u *User) IsActive() bool {
	return u.Active
}

func (u *User) SetEmail(x string) {
	u.Email = x
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) IsEmailConfirmed() bool {
	return u.EmailConfirmed
}

func (u *User) SetIsEmailConfirmed(x bool) {
	u.EmailConfirmed = x
}

func (u *User) SetUsername(x string) {
	u.Username = x
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) SetLastLogin(x time.Time) {
	u.LastLogin = x
}

func (u *User) GetLastLogin() time.Time {
	return u.LastLogin
}

func (u *User) GetData() (interface{}, kit.Error) {
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

func (u *User) SetData(x interface{}) kit.Error {
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

func (u *User) SetCreatedAt(x time.Time) {
	u.CreatedAt = x
}

func (u *User) GetCreatedAt() time.Time {
	return u.CreatedAt
}

func (u *User) SetUpdatedAt(x time.Time) {
	u.UpdatedAt = x
}

func (u *User) GetUpdatedAt() time.Time {
	return u.UpdatedAt
}

/**
 * RBAC methods.
 */

func (u *User) GetRoles() []kit.Role {
	slice := make([]kit.Role, 0)
	for _, r := range u.Roles {
		slice = append(slice, r)
	}
	return slice
}

func (u *User) AddRole(r kit.Role) {
	if u.Roles == nil {
		u.Roles = make([]*Role, 0)
	}
	if !u.HasRole(r) {
		u.Roles = append(u.Roles, r.(*Role))
	}
}

func (u *User) RemoveRole(r kit.Role) {
	if u.Roles == nil {
		return
	}

	for i := 0; i < len(u.Roles); i++ {
		if u.Roles[i].Name == r.GetName() {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
		}
	}
}

func (u *User) ClearRoles() {
	u.Roles = make([]*Role, 0)
}

func (u *User) HasRole(r kit.Role) bool {
	return u.HasRoleStr(r.GetName())
}

func (u *User) HasRoleStr(role string) bool {
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

type IntUser struct {
	db.BaseIntModel
	User
}

/**
 * UserProfile.
 */

type UserProfile struct {
	UserModel
}

func (p UserProfile) Collection() string {
	return "user_profiles"
}

func (p *UserProfile) GetID() string {
	return p.GetUserID()
}

func (p *UserProfile) SetID(rawId string) error {
	return p.SetUserID(rawId)
}

type IntUserProfile struct {
	UserProfile
	IntUserModel
}

func (p *IntUserProfile) GetID() string {
	return p.GetUserID()
}

func (p *IntUserProfile) SetID(rawId string) error {
	return p.SetUserID(rawId)
}

/**
 * Token.
 */

type Token struct {
	UserModel

	Token     string    `db:"primary-key"`
	Type      string    `db:"not-null"`
	ExpiresAt time.Time `db:"ignore-zero"`
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
 * Session
 */

type Session struct {
	UserModel

	Token string `db:"primary-key;max:150"`
	Typ   string `db:"not-null;max:100"`

	StartedAt  time.Time `db:"not-null"`
	ValidUntil time.Time `db:"not-null"`
}

func (b Session) Collection() string {
	return "sessions"
}

func (s Session) GetID() string {
	return s.Token
}

func (s *Session) SetID(x string) error {
	s.Token = x
	return nil
}

func (s *Session) GetType() string {
	return s.Typ
}

func (s *Session) SetType(x string) {
	s.Typ = x
}

func (s *Session) SetToken(x string) {
	s.Token = x
}

func (s *Session) GetToken() string {
	return s.Token
}

func (s *Session) SetStartedAt(x time.Time) {
	s.StartedAt = x
}

func (s *Session) GetStartedAt() time.Time {
	return s.StartedAt
}

func (s *Session) SetValidUntil(x time.Time) {
	s.ValidUntil = x
}

func (s *Session) GetValidUntil() time.Time {
	return s.ValidUntil
}

func (s *Session) IsGuest() bool {
	return s.UserID == ""
}

type IntUserSession struct {
	IntUserModel
	Session
}

func (s *IntUserSession) IsGuest() bool {
	return s.UserID == 0
}

/**
 * Role.
 */

type Role struct {
	Name        string        `db:"primary-key;max:200"`
	Permissions []*Permission `db:"m2m"`
}

func (r Role) Collection() string {
	return "user_roles"
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
	Name string `db:"primary-key;max:200"`
}

func (r Permission) Collection() string {
	return "user_permissions"
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
