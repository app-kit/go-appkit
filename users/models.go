package users

import (
	"encoding/json"
	"time"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-reflector"

	kit "github.com/app-kit/go-appkit"
)

/**
 * Extendable models that are related to a user.
 */

type StrUserModel struct {
	User   *UserStrId
	UserId string `db:"max:255"`
}

func (m *StrUserModel) GetUserId() interface{} {
	return m.UserId
}

func (m *StrUserModel) SetUserId(id interface{}) error {
	if strId, ok := id.(string); ok {
		m.UserId = strId
		return nil
	}

	convertedId, err := reflector.R(id).ConvertTo("")
	if err != nil {
		return err
	}

	m.UserId = convertedId.(string)
	return nil
}

func (m *StrUserModel) GetUser() kit.User {
	if m.User != nil {
		return m.User
	}
	return nil
}

func (m *StrUserModel) SetUser(u kit.User) {
	if u == nil {
		m.User = nil
		m.UserId = ""
	} else {
		m.User = u.(*UserStrId)
		m.SetUserId(u.GetId())
	}
}

type IntUserModel struct {
	User   *UserIntId
	UserId uint64 `db:""`
}

func (m *IntUserModel) GetUserId() interface{} {
	return m.UserId
}

func (m *IntUserModel) SetUserId(id interface{}) error {
	if intId, ok := id.(uint64); ok {
		m.UserId = intId
		return nil
	}

	convertedId, err := reflector.R(id).ConvertTo(uint64(0))
	if err != nil {
		return err
	}

	m.UserId = convertedId.(uint64)
	return nil
}

func (m *IntUserModel) GetUser() kit.User {
	if m.User != nil {
		return m.User
	}
	return nil
}

func (m *IntUserModel) SetUser(u kit.User) {
	if u == nil {
		m.User = nil
		m.UserId = 0
	} else {
		m.User = u.(*UserIntId)
		m.SetUserId(u.GetId())
	}
}

/**
 * User.
 */

type User struct {
	db.TimeStampedModel

	Active bool

	Username string `db:"unique;required"`
	Email    string `db:"unique;required"`

	EmailConfirmed bool

	LastLogin time.Time `db:"ignore-zero"`

	Data string `db:"ignore-zero;max:10000"`

	Profile interface{} `db:"-"`

	Roles []*Role `db:"m2m;"`
}

func (u User) Collection() string {
	return "users"
}

// Implement User interface.

func (a User) GetProfile() kit.UserProfile {
	if a.Profile == nil {
		return nil
	}
	return a.Profile.(kit.UserProfile)
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

func (u *User) GetData() (interface{}, apperror.Error) {
	if u.Data == "" {
		return nil, nil
	}
	var data interface{}
	err := json.Unmarshal([]byte(u.Data), &data)
	if err != nil {
		return nil, apperror.Wrap(err, "json_marshal_error")
	}
	return data, nil
}

func (u *User) SetData(x interface{}) apperror.Error {
	js, err := json.Marshal(x)
	if err != nil {
		return apperror.Wrap(err, "json_marshal_error")
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

func (u *User) GetRoles() []string {
	slice := make([]string, 0)
	for _, r := range u.Roles {
		slice = append(slice, r.Name)
	}
	return slice
}

func (u *User) SetRoles(roles []string) {
	newRoles := make([]*Role, 0)
	for _, r := range roles {
		newRoles = append(newRoles, &Role{Name: r})
	}
	u.Roles = newRoles
}

func (u *User) AddRole(roles ...string) {
	for _, role := range roles {
		if !u.HasRole(role) {
			u.Roles = append(u.Roles, &Role{Name: role})
		}
	}
}

func (u *User) RemoveRole(roles ...string) {
	newRoles := make([]*Role, 0)

	for _, role := range u.Roles {
		keep := true
		for _, roleToRemove := range roles {
			if role.Name == roleToRemove {
				keep = false
				break
			}
		}
		if keep {
			newRoles = append(newRoles, role)
		}
	}

	u.Roles = newRoles
}

func (u *User) ClearRoles() {
	u.Roles = make([]*Role, 0)
}

func (u *User) HasRole(roles ...string) bool {
	for _, role := range u.Roles {
		for _, matchRole := range roles {
			if role.Name == matchRole {
				return true
			}
		}
	}

	return false
}

func (u *User) HasPermission(perms ...string) bool {
	for _, r := range u.Roles {
		if r.HasPermission(perms...) {
			return true
		}
	}

	return false
}

type UserStrId struct {
	User
	db.StrIdModel
}

// Ensure UserStrId implements kit.User interface.
var _ kit.User = (*UserStrId)(nil)

func (u *UserStrId) SetProfile(p kit.UserProfile) {
	p.SetUser(u)
	u.Profile = p
}

type UserIntId struct {
	User
	db.IntIdModel
}

// Ensure UserIntId implements kit.User interface.
var _ kit.User = (*UserStrId)(nil)

func (u *UserIntId) SetProfile(p kit.UserProfile) {
	p.SetUser(u)
	u.Profile = p
}

/**
 * UserProfile.
 */

type StrIdUserProfile struct {
	db.StrIdModel
}

func (p StrIdUserProfile) Collection() string {
	return "user_profiles"
}

func (p StrIdUserProfile) GetUserId() interface{} {
	return p.Id
}

func (p *StrIdUserProfile) SetUserId(id interface{}) error {
	return p.SetId(id)
}

func (p StrIdUserProfile) GetUser() kit.User {
	return nil
}

func (p StrIdUserProfile) SetUser(user kit.User) {
	p.SetUserId(user.GetId())
}

type IntIdUserProfile struct {
	db.IntIdModel
}

func (p IntIdUserProfile) Collection() string {
	return "user_profiles"
}

func (p IntIdUserProfile) GetUserId() interface{} {
	return p.Id
}

func (p *IntIdUserProfile) SetUserId(id interface{}) error {
	return p.SetId(id)
}

func (p IntIdUserProfile) GetUser() kit.User {
	return nil
}

func (p IntIdUserProfile) SetUser(user kit.User) {
	p.SetUserId(user.GetId())
}

/**
 * Token.
 */

type Token struct {
	StrUserModel

	Token     string    `db:"primary-key"`
	Type      string    `db:"required"`
	ExpiresAt time.Time `db:"ignore-zero"`
}

// Ensure that Token implements Token interface.
var _ kit.UserToken = (*Token)(nil)

func (t *Token) Collection() string {
	return "user_tokens"
}

func (t *Token) GetId() interface{} {
	return t.Token
}

func (t *Token) GetStrId() string {
	return t.Token
}

func (t *Token) SetId(id interface{}) error {
	t.Token = id.(string)
	return nil
}

func (t *Token) SetStrId(id string) error {
	t.Token = id
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
	Token string `db:"primary-key;max:150"`
	Type  string `db:"max:10"`

	StartedAt  time.Time `db:"required"`
	ValidUntil time.Time `db:"required"`
}

func (b Session) Collection() string {
	return "sessions"
}

func (s Session) GetId() interface{} {
	return s.Token
}

func (s *Session) SetId(x interface{}) error {
	s.Token = x.(string)
	return nil
}

func (s Session) GetStrId() string {
	return s.Token
}

func (s *Session) SetStrId(x string) error {
	s.Token = x
	return nil
}

func (s *Session) GetType() string {
	return s.Type
}

func (s *Session) SetType(x string) {
	s.Type = x
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

type StrUserSession struct {
	StrUserModel
	Session
}

func (s StrUserSession) IsAnonymous() bool {
	return s.UserId == ""
}

type IntUserSession struct {
	IntUserModel
	Session
}

func (s *IntUserSession) IsAnonymous() bool {
	return s.UserId == 0
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

func (r Role) GetId() interface{} {
	return r.Name
}

func (r *Role) SetId(n interface{}) error {
	r.Name = n.(string)
	return nil
}

func (r Role) GetStrId() string {
	return r.Name
}

func (r *Role) SetStrId(n string) error {
	r.Name = n
	return nil
}

func (r *Role) GetPermissions() []string {
	perms := make([]string, 0)
	for _, p := range r.Permissions {
		perms = append(perms, p.Name)
	}
	return perms
}

func (r *Role) SetPermissions(perms []string) {
	permList := make([]*Permission, 0)
	for _, p := range perms {
		permList = append(permList, &Permission{Name: p})
	}
	r.Permissions = permList
}

func (r *Role) AddPermission(perms ...string) {
	for _, perm := range perms {
		if !r.HasPermission(perm) {
			r.Permissions = append(r.Permissions, &Permission{Name: perm})
		}
	}
}

func (r *Role) RemovePermission(perms ...string) {
	var newPerms []*Permission

	for _, perm := range r.Permissions {
		matched := false
		for _, matchPerm := range perms {
			if perm.Name == matchPerm {
				matched = true
				break
			}
		}
		if !matched {
			newPerms = append(newPerms, perm)
		}
	}

	r.Permissions = newPerms
}

func (r *Role) ClearPermissions() {
	r.Permissions = make([]*Permission, 0)
}

func (r *Role) HasPermission(perms ...string) bool {
	for _, p := range r.Permissions {
		for _, matchPerm := range perms {
			if p.Name == matchPerm {
				return true
			}
		}
	}

	return false
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

func (p Permission) GetId() interface{} {
	return p.Name
}

func (p *Permission) SetId(n interface{}) error {
	p.Name = n.(string)
	return nil
}

func (p Permission) GetStrId() string {
	return p.Name
}

func (p *Permission) SetStrId(n string) error {
	p.Name = n
	return nil
}
